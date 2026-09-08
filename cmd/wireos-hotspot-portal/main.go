package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/os-vector/wired/mods"
)

func main() {
	wifi := mods.NewWifiSetup()
	if err := wifi.Load(); err != nil {
		panic(err)
	}

	wiredURL, err := url.Parse("http://127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	wiredUI := httputil.NewSingleHostReverseProxy(wiredURL)
	wiredUI.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Giao diện chính chưa sẵn sàng. Hãy thử lại sau.", http.StatusBadGateway)
	}

	http.HandleFunc("/api/mods/WifiSetup/", wifi.HTTP)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Keep the complete wired UI (menus, icons, controls and Bot Settings)
		// available on the hotspot's standard HTTP address. WifiSetup owns its
		// explicit /wifi and API routes; every other path is proxied to :8080.
		wiredUI.ServeHTTP(w, r)
	})

	fmt.Println("wireos-hotspot-portal listening on :8081 (full UI proxy → :8080)")
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
