package sqlite

import (
	"database/sql"
	"log"
	"time"

	"github.com/StableSteady/window-watcher/util"

	_ "github.com/mattn/go-sqlite3"
)

const (
	createProcInfoTableQuery       = `CREATE TABLE IF NOT EXISTS process (id INTEGER PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL, desc TEXT NOT NULL, track INTEGER NOT NULL)`
	createProcDataTableQuery       = `CREATE TABLE IF NOT EXISTS data (id INTEGER NOT NULL, time TIMESTAMP NOT NULL PRIMARY KEY)`
	getProcessTimeInDescOrderQuery = `SELECT desc, path, count(*)
										FROM data 
										INNER JOIN process ON process.id = data.id 
										GROUP BY data.id 
										ORDER BY count(*) DESC`
	deleteProcessByPathQuery = `DELETE
										FROM data 
										WHERE ROWID in (
											SELECT a.ROWID FROM data a
											INNER JOIN process b ON a.id = b.id
											WHERE path = ?
										)`
	getExclusionsQuery = `SELECT path
										FROM process
										WHERE track = 0`
	updateExclusionQuery = `UPDATE process
										SET track = ?
										WHERE path = ?`
	addExclusionQuery = `INSERT INTO process (name, path, desc, track) 
										VALUES (?, ?, ?, 0)`
	getTrackStatusByPathQuery = `SELECT track
										FROM process
										WHERE path = ?`
	procInfoStmtQuery = `INSERT INTO process(name, path, desc, track)
										VALUES(?, ?, ?, ?)`
	procDataStmtQuery = `INSERT INTO data
										VALUES(?, ?)`
	searchInProcInfoQuery = `SELECT id, track
										FROM process 
										WHERE path = ?`
)

var (
	db *sql.DB
	getProcessTimeInDescOrder,
	deleteProcessByPath,
	getExclusions,
	updateExclusion,
	addExclusion,
	getTrackStatusByPath,
	procInfoStmt,
	procDataStmt,
	searchInProcInfo *sql.Stmt
)

func prepareStmt() {
	var err error
	getProcessTimeInDescOrder, err = db.Prepare(getProcessTimeInDescOrderQuery)
	if err != nil {
		log.Fatal(err)
	}

	deleteProcessByPath, err = db.Prepare(deleteProcessByPathQuery)
	if err != nil {
		log.Fatal(err)
	}

	getExclusions, err = db.Prepare(getExclusionsQuery)
	if err != nil {
		log.Fatal(err)
	}

	updateExclusion, err = db.Prepare(updateExclusionQuery)
	if err != nil {
		log.Fatal(err)
	}

	addExclusion, err = db.Prepare(addExclusionQuery)
	if err != nil {
		log.Fatal(err)
	}

	getTrackStatusByPath, err = db.Prepare(getTrackStatusByPathQuery)
	if err != nil {
		log.Fatal(err)
	}

	procInfoStmt, err = db.Prepare(procInfoStmtQuery)
	if err != nil {
		log.Fatal(err)
	}

	procDataStmt, err = db.Prepare(procDataStmtQuery)
	if err != nil {
		log.Fatal(err)
	}

	searchInProcInfo, err = db.Prepare(searchInProcInfoQuery)
	if err != nil {
		log.Fatal(err)
	}
}

func CloseDB() {
	getProcessTimeInDescOrder.Close()
	deleteProcessByPath.Close()
	getExclusions.Close()
	updateExclusion.Close()
	addExclusion.Close()
	getTrackStatusByPath.Close()
	procInfoStmt.Close()
	procDataStmt.Close()
	searchInProcInfo.Close()
	db.Close()
}

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./ww.db?_journal=off")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createProcInfoTableQuery)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createProcDataTableQuery)
	if err != nil {
		log.Fatal(err)
	}

	prepareStmt()
}

func GetExclusions() (paths []string, err error) {
	rows, err := getExclusions.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		err = rows.Scan(&path)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func UpdateExclusion(track int, path string) error {
	_, err := updateExclusion.Exec(1, path)
	return err
}

func GetTrackStatusByPath(path string) (int, error) {
	var track int
	err := getTrackStatusByPath.QueryRow(path).Scan(&track)
	return track, err
}

func AddExclusion(processName, desc, path string) error {
	_, err := addExclusion.Exec(processName, desc, path)
	return err
}

func GetProcessTimeInDescOrder() (list [][]string, err error) {
	rows, err := getProcessTimeInDescOrder.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, path string
		var duration int
		err = rows.Scan(&name, &path, &duration)
		if err != nil {
			return nil, err
		}
		list = append(list, []string{name, util.SecondsToHuman(duration), path})
	}
	return
}

func DeleteProcessByPath(path string) error {
	_, err := deleteProcessByPath.Exec(path)
	return err
}

func DeleteDB() {
	_, err := db.Exec("DELETE FROM data")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("DELETE FROM process")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("VACUUM")
	if err != nil {
		log.Fatal(err)
	}
}

func SearchInProcInfo(path string, id, track *int) error {
	return searchInProcInfo.QueryRow(path).Scan(id, track)
}

func InsertProcessData(filename, path, desc string, track bool) error {
	_, err := procInfoStmt.Exec(filename, path, desc, true)
	return err
}

func InsertProcessTime(id int) error {
	_, err := procDataStmt.Exec(id, time.Now())
	return err
}
