package main

import (
	"github.com/StableSteady/window-watcher/gui"
	"github.com/StableSteady/window-watcher/sqlite"
	"github.com/StableSteady/window-watcher/window"
)

func main() {
	go window.Watch()
	gui.Start()
	sqlite.CloseDB()
}
