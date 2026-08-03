package main

import (
	"fmt"
	"net/http"

	"github.com/os-vector/wired/mods"
	"github.com/os-vector/wired/vars"
)

var EnabledMods []vars.Modification = []vars.Modification{
	mods.NewFreqChange(),
	mods.NewWakeEngine(),
	mods.NewWakeWordPV(),
	mods.NewAutoUpdate(),
	mods.NewSensitivityPV(),
	mods.NewJdocSettings(),
	mods.NewFaces(),
	mods.NewXiaozhi(),
	mods.NewControl(),
	mods.NewAlexa(),
	mods.NewPhotos(),
	mods.NewStimStats(),
	mods.NewJournalLogs(),
	mods.NewChess(),
	mods.NewXiangqi(),
	mods.NewCaro(),
	mods.NewConnect4(),
	mods.NewReversi(),
	mods.NewCheckers(),
	mods.NewGo9(),
	mods.NewWebLocale(),
}

func main() {
	vars.EnabledMods = EnabledMods
	vars.InitMods()
	startweb()
}

func startweb() {
	fmt.Println("starting web at ports 80 and 8080")
	fs := http.FileServer(http.Dir("/etc/wired/webroot"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// no mno non o caching
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		fs.ServeHTTP(w, r)
	})
	// Same DefaultServeMux on both ports. Prefer :8080 as the blocking
	// listener so a bind failure on privileged :80 still leaves the UI up.
	go func() {
		if err := http.ListenAndServe(":80", nil); err != nil {
			fmt.Println("wired listen :80 failed:", err)
		}
	}()
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("wired listen :8080 failed:", err)
	}
}
