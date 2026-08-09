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
	mods.NewFreeTime(),
	mods.NewBattery(),
	mods.NewAlexa(),
	mods.NewPhotos(),
	mods.NewStimStats(),
	mods.NewPetting(),
	mods.NewJournalLogs(),
	mods.NewChess(),
	mods.NewXiangqi(),
	mods.NewCaro(),
	mods.NewConnect4(),
	mods.NewReversi(),
	mods.NewCheckers(),
	mods.NewGo9(),
	mods.NewTicTacToe(),
	mods.NewSudoku(),
	mods.NewG2048(),
	mods.NewMinesweeper(),
	mods.NewMemory(),
	mods.NewBattleship(),
	mods.NewWordle(),
	mods.NewHangman(),
	mods.NewTrivia(),
	mods.NewGuessNum(),
	mods.NewSimon(),
	mods.NewUno(),
	mods.NewBlackjackWeb(),
	mods.NewPoker(),
	mods.NewWebLocale(),
}

func main() {
	vars.EnabledMods = EnabledMods
	vars.InitMods()
	startweb()
}

func startweb() {
	fmt.Println("starting web at ports 80, 8080 (HTTP) and 8443 (HTTPS for mic)")
	fs := http.FileServer(http.Dir("/etc/wired/webroot"))
	mux := http.DefaultServeMux
	// Root file server — InitMods already registered /api/mods/... on DefaultServeMux.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fs.ServeHTTP(w, r)
	})

	startHTTPS(mux)

	go func() {
		if err := http.ListenAndServe(":80", nil); err != nil {
			fmt.Println("wired listen :80 failed:", err)
		}
	}()
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("wired listen :8080 failed:", err)
	}
}
