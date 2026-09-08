package main

import (
	"fmt"
	"net/http"

	"github.com/os-vector/wired/mods"
)

func main() {
	wifi := mods.NewWifiSetup()
	if err := wifi.Load(); err != nil {
		panic(err)
	}

	http.HandleFunc("/api/mods/WifiSetup/", wifi.HTTP)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/wifi", http.StatusFound)
	})

	fmt.Println("wireos-hotspot-portal listening on :8081")
	cors := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	if err := http.ListenAndServe(":8081", mods.WrapCaptive(cors)); err != nil {
		panic(err)
	}
}
