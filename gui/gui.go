package gui

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/StableSteady/window-watcher/sqlite"
	"github.com/StableSteady/window-watcher/window"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tadvi/winc"
)

type Item struct {
	T []string
}

func (item Item) Text() []string  { return item.T }
func (item Item) ImageIndex() int { return 0 }

var (
	mainWindow      *winc.Form
	dock            *winc.SimpleDock
	aboutButton     *winc.PushButton
	exclusionButton *winc.PushButton
	optionButton    *winc.PushButton
	statButton      *winc.PushButton
	quitButton      *winc.PushButton
)

func wndOnClose(arg *winc.Event) {
	winc.Exit()
}

func hideButtons() {
	aboutButton.Hide()
	exclusionButton.Hide()
	optionButton.Hide()
	statButton.Hide()
	quitButton.Hide()
}

func showButtons() {
	aboutButton.Show()
	exclusionButton.Show()
	optionButton.Show()
	statButton.Show()
	quitButton.Show()
}

func init() {
	mainWindow = winc.NewForm(nil)
	dock = winc.NewSimpleDock(mainWindow)
	mainWindow.SetLayout(dock)
	mainWindow.SetSize(400, 200) // (width, height)
	mainWindow.SetText("Window Watcher")
	mainWindow.OnClose().Bind(wndOnClose)
	mainWindow.EnableSizable(false)
	mainWindow.EnableMaxButton(false)

	aboutButton = winc.NewPushButton(mainWindow)
	aboutButton.SetText("About")
	exclusionButton = winc.NewPushButton(mainWindow)
	exclusionButton.SetText("Exclusions")
	optionButton = winc.NewPushButton(mainWindow)
	optionButton.SetText("Options")
	statButton = winc.NewPushButton(mainWindow)
	statButton.SetText("Stats")
	quitButton = winc.NewPushButton(mainWindow)
	quitButton.SetText("Quit")

	dock.Dock(aboutButton, winc.Left)
	dock.Dock(exclusionButton, winc.Top)
	dock.Dock(optionButton, winc.Right)
	dock.Dock(statButton, winc.Fill)
	dock.Dock(quitButton, winc.Bottom)

	exclusionButton.OnClick().Bind(func(arg *winc.Event) {
		mainWindow.SetSize(400, 600)
		hideButtons()
		list := winc.NewListView(mainWindow)
		list.AddColumn("Path", 400)
		paths, err := sqlite.GetExclusions()
		if err != nil {
			log.Fatal(err)
		}
		for _, path := range paths {
			list.AddItem(&Item{[]string{path}})
		}
		dock.Dock(list, winc.Fill)
		edit := winc.NewEdit(mainWindow)

		addButton := winc.NewPushButton(mainWindow)
		addButton.SetText("Add")

		backButton := winc.NewPushButton(mainWindow)
		backButton.SetText("Back")

		deleteButton := winc.NewPushButton(mainWindow)
		deleteButton.SetText("Delete")

		dock.Dock(backButton, winc.Bottom)
		dock.Dock(deleteButton, winc.Bottom)
		dock.Dock(addButton, winc.Bottom)
		dock.Dock(edit, winc.Bottom)

		backButton.OnClick().Bind(func(arg *winc.Event) {
			list.Close()
			edit.Close()
			addButton.Close()
			backButton.Close()
			deleteButton.Close()
			mainWindow.SetSize(400, 200)
			showButtons()
		})
		deleteButton.OnClick().Bind(func(arg *winc.Event) {
			path := list.SelectedItem().Text()[0]
			err := sqlite.UpdateExclusion(1, path)
			if err != nil {
				log.Fatal(err)
			}
			list.DeleteItem(list.SelectedItem())
		})
		addButton.OnClick().Bind(func(arg *winc.Event) {
			track, err := sqlite.GetTrackStatusByPath(edit.Text())
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					desc, err := window.GetDescriptionFromPath(edit.Text())
					if err != nil {
						log.Fatal(err)
					}
					pathSlice := strings.Split(edit.Text(), "/")
					processName := pathSlice[len(pathSlice)-1]
					err = sqlite.AddExclusion(processName, desc, edit.Text())
					if err != nil {
						log.Fatal(err)
					}
				} else {
					log.Fatal(err)
				}
			} else {
				if track == 1 {
					err := sqlite.UpdateExclusion(0, edit.Text())
					if err != nil {
						log.Fatal(err)
					}
					list.AddItem(&Item{[]string{edit.Text()}})
				}
			}
		})
	})
	statButton.OnClick().Bind(func(arg *winc.Event) {
		mainWindow.SetSize(400, 600)
		hideButtons()
		list := winc.NewListView(mainWindow)
		list.EnableEditLabels(false)
		list.EnableFullRowSelect(true)
		dock.Dock(list, winc.Fill)
		list.AddColumn("Process", 200)
		list.AddColumn("Time", 200)
		items, err := sqlite.GetProcessTimeInDescOrder()
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range items {
			list.AddItem(&Item{item})
		}
		panel := winc.NewPanel(mainWindow)
		dock.Dock(panel, winc.Bottom)
		panelDock := winc.NewSimpleDock(panel)
		panel.SetLayout(panelDock)
		deleteButton := winc.NewPushButton(panel)
		deleteButton.SetText("Delete")
		backButton := winc.NewPushButton(panel)
		backButton.SetText("Back")
		panelDock.Dock(deleteButton, winc.Left)
		panelDock.Dock(backButton, winc.Fill)

		deleteButton.OnClick().Bind(func(arg *winc.Event) {
			path := list.SelectedItem().Text()[2]
			err := sqlite.DeleteProcessByPath(path)
			if err != nil {
				log.Println(err)
			}
			list.DeleteItem(list.SelectedItem())
		})

		backButton.OnClick().Bind(func(arg *winc.Event) {
			list.Close()
			panel.Close()
			deleteButton.Close()
			backButton.Close()
			mainWindow.SetSize(400, 200)
			showButtons()
		})
	})
	aboutButton.OnClick().Bind(func(arg *winc.Event) {
		dialog := winc.NewDialog(mainWindow)
		dialog.SetText("About")
		text := winc.NewLabel(dialog)
		text.SetSize(text.Parent().Size())
		text.SetText("Made by Avijeet Maurya.")
		dialog.Show()
		dialog.OnClose().Bind(func(arg *winc.Event) { dialog.Close() })
	})
	optionButton.OnClick().Bind(func(arg *winc.Event) {
		hideButtons()
		deleteButton := winc.NewPushButton(mainWindow)
		deleteButton.SetText("Delete Database")
		deleteButton.OnClick().Bind(func(arg *winc.Event) {
			sqlite.DeleteDB()
		})
		backButton := winc.NewPushButton(mainWindow)
		backButton.SetText("Back")
		backButton.OnClick().Bind(func(arg *winc.Event) {
			deleteButton.Close()
			backButton.Close()
			showButtons()
		})
		dock.Dock(deleteButton, winc.Fill)
		dock.Dock(backButton, winc.Bottom)
	})
	quitButton.OnClick().Bind(wndOnClose)
}

func Start() {
	mainWindow.Show()
	winc.RunMainLoop()
}
