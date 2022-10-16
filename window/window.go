package window

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/StableSteady/window-watcher/sqlite"

	"golang.org/x/sys/windows"

	_ "github.com/mattn/go-sqlite3"
)

const (
	Ready = iota
	Stop
)

// get slice containing language strings
func verQueryValueTranslations(block []byte) ([]string, error) {
	var offset uintptr
	var length uint32
	blockStart := unsafe.Pointer(&block[0])
	err := windows.VerQueryValue(
		blockStart,
		`\VarFileInfo\Translation`,
		unsafe.Pointer(&offset),
		&length,
	)
	if err != nil {
		return nil, err
	}

	start := int(offset) - int(uintptr(blockStart))
	end := start + int(length)
	if start < 0 || start >= len(block) || end < start || end > len(block) {
		return nil, errors.New("verQueryValueTranslations: Invalid start or end position for item string")
	}

	data := block[start:end]
	// each translation consists of a 16-bit language ID and a 16-bit code page
	// ID, so each entry has 4 bytes
	if len(data)%4 != 0 {
		return nil, errors.New("verQueryValueTranslations: incorrect number of bytes")
	}

	trans := make([]string, len(data)/4)
	for i := range trans {
		t := data[i*4 : (i+1)*4]
		// handle endianness of the 16-bit values
		t[0], t[1] = t[1], t[0]
		t[2], t[3] = t[3], t[2]
		trans[i] = fmt.Sprintf("%x", t)
	}
	return trans, nil
}

// get executable information in the item field
func verQueryValueString(block []byte, translation, item string) (string, error) {
	var offset uintptr
	var utf16Length uint32
	blockStart := unsafe.Pointer(&block[0])
	id := `\StringFileInfo\` + translation + `\` + item
	err := windows.VerQueryValue(
		blockStart,
		id,
		unsafe.Pointer(&offset),
		&utf16Length,
	)
	if err != nil {
		return "", err
	}

	start := int(offset) - int(uintptr(blockStart))
	end := start + int(2*utf16Length)
	if start < 0 || start >= len(block) || end < start || end > len(block) {
		return "", errors.New("verQueryValueString: Invalid start or end position for item string")
	}

	data := block[start:end]
	u16 := make([]uint16, utf16Length)
	for i := range u16 {
		u16[i] = uint16(data[i*2+1])<<8 | uint16(data[i*2+0])
	}
	return windows.UTF16ToString(u16), nil
}

func getFilePathAndNameFromProcessID(processID uint32) (string, string, error) {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, processID)
	if err != nil {
		return "", "", fmt.Errorf("error in OpenProcess: %v", err)
	}

	var hMod windows.Handle
	var cbNeeded uint32
	err = windows.EnumProcessModulesEx(hProcess, &hMod, 4, &cbNeeded, windows.LIST_MODULES_ALL)
	if err != nil {
		return "", "", fmt.Errorf("error in EnumProcessModulesEx: %v", err)
	}

	basename := make([]uint16, 260)
	err = windows.GetModuleBaseName(hProcess, hMod, &basename[0], 260)
	if err != nil {
		return "", "", fmt.Errorf("error in GetModuleBaseName: %v", err)
	}

	fileName := windows.UTF16ToString(basename)
	if fileName == "" {
		return "", "", errors.New("empty filename")
	}

	var filePathLen uint32 = 300
	filePath := make([]uint16, filePathLen)
	err = windows.QueryFullProcessImageName(hProcess, 0, &filePath[0], &filePathLen)
	if err != nil {
		return "", "", fmt.Errorf("error in QueryFullProcessImageName: %v", err)
	}

	return windows.UTF16ToString(filePath), fileName, nil
}

func GetDescriptionFromPath(path string) (string, error) {
	versionInfoSize, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil {
		return "", fmt.Errorf("error in GetFileVersionInfoSize: %v", err)
	}

	versionInfo := make([]uint8, versionInfoSize)
	err = windows.GetFileVersionInfo(path, 0, versionInfoSize, unsafe.Pointer(&versionInfo[0]))
	if err != nil {
		return "", fmt.Errorf("error in GetFileVersionInfo: %v", err)
	}

	languages, err := verQueryValueTranslations(versionInfo)
	if err != nil {
		return "", fmt.Errorf("error in verQueryValueTranslations: %v", err)
	}

	desc, err := verQueryValueString(versionInfo, languages[0], "FileDescription")
	if err != nil {
		return "", fmt.Errorf("error in verQueryValueString: %v", err)
	}
	return desc, nil
}

func getProcessInfo(processID uint32) (string, string, string, error) {
	filePathName, fileName, err := getFilePathAndNameFromProcessID(processID)
	if err != nil {
		return "", "", "", err
	}

	desc, err := GetDescriptionFromPath(filePathName)
	if err != nil {
		return "", "", "", err
	}

	return fileName, desc, filePathName, nil
}

func getCurrentProcess() (string, uint32, error) {
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		return "", 0, nil
	}

	var processID uint32
	windows.GetWindowThreadProcessId(hwnd, &processID)
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, processID)
	if err != nil {
		return "", 0, fmt.Errorf("error in OpenProcess: %v", err)
	}

	var filePathLen uint32 = 300
	filePath := make([]uint16, filePathLen)
	err = windows.QueryFullProcessImageName(hProcess, 0, &filePath[0], &filePathLen)
	if err != nil {
		return "", 0, fmt.Errorf("error in QueryFullProcessImageName: %v", err)
	}

	return windows.UTF16ToString(filePath), processID, nil
}

func Watch() {
	for {
		time.Sleep(time.Second)
		path, processID, err := getCurrentProcess()
		//empty string is returned when hwnd of foreground window is null
		if path == "" {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		var id, track int
		err = sqlite.SearchInProcInfo(path, &id, &track)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				filename, desc, _, err := getProcessInfo(processID)
				if filename == "" {
					continue
				}
				if err != nil {
					log.Fatal(err)
				}
				err = sqlite.InsertProcessData(filename, path, desc, true)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal(err)
			}
		}
		if track == 0 {
			continue
		}
		err = sqlite.InsertProcessTime(id)
		if err != nil {
			log.Fatal(err)
		}
	}
}
