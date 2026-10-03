package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/os-vector/wired/mods"
)

var portalAPIAllowlist = []string{
	"/api/mods/FreqChange/",
	"/api/mods/WakeEngine/",
	"/api/mods/WakeWordPV/",
	"/api/mods/SensitivityPV/",
	"/api/mods/JdocSettings/",
	"/api/mods/Faces/",
	"/api/mods/Photos/",
	"/api/mods/StimStats/",
	"/api/mods/Petting/",
	"/api/mods/Alexa/",
	"/api/mods/Xiaozhi/",
	"/api/mods/Control/",
	"/api/extra/restartvic",
}

func portalAPIAllowed(path string) bool {
	for _, prefix := range portalAPIAllowlist {
		if (strings.HasSuffix(prefix, "/") && strings.HasPrefix(path, prefix)) ||
			(!strings.HasSuffix(prefix, "/") && path == prefix) {
			return true
		}
	}
	return false
}

func main() {
	wifi := mods.NewWifiSetup()
	if err := wifi.Load(); err != nil {
		panic(err)
	}

	wiredURL, err := url.Parse("http://127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	wiredAPI := httputil.NewSingleHostReverseProxy(wiredURL)
	wiredAPI.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Dịch vụ cài đặt chính chưa sẵn sàng.", http.StatusBadGateway)
	}

	http.HandleFunc("/api/mods/WifiSetup/", wifi.HTTP)
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if !portalAPIAllowed(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		wiredAPI.ServeHTTP(w, r)
	})
	http.Handle("/portal-assets/", http.StripPrefix(
		"/portal-assets/",
		http.FileServer(http.Dir("/etc/wired/webroot")),
	))
	http.Handle("/hotspot/", http.StripPrefix(
		"/hotspot/",
		http.FileServer(http.Dir("/etc/wired/hotspot-webroot")),
	))
	http.Handle("/portal/", http.StripPrefix(
		"/portal/",
		http.FileServer(http.Dir("/etc/wired/hotspot-webroot")),
	))
	http.HandleFunc("/portal", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/portal/", http.StatusFound)
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
