package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/os-vector/wired/mods"
)

func main() {
	wifi := mods.NewWifiSetup()
	if err := wifi.Load(); err != nil {
		panic(err)
	}

	http.HandleFunc("/api/mods/WifiSetup/", wifi.HTTP)
	http.HandleFunc("/api/hotspot/test-animation", func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if err != nil || ip == nil || !ip.IsLoopback() {
			http.Error(w, "localhost only", http.StatusForbidden)
			return
		}
		state := r.FormValue("state")
		if state != "trying" && state != "ok" && state != "fail" {
			http.Error(w, "state must be trying, ok, or fail", http.StatusBadRequest)
			return
		}
		mods.PlayWifiStatusAnimation(state)
		w.WriteHeader(http.StatusAccepted)
	})
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
