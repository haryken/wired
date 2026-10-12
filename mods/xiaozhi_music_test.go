package mods

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func postXiaozhi(t *testing.T, endpoint string, form url.Values) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/"+endpoint+"?"+form.Encode(), nil)
	w := httptest.NewRecorder()
	NewXiaozhi().HTTP(w, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

func TestXiaozhiSetMusicSavesSourceAndURL(t *testing.T) {
	withTempXiaozhiPaths(t)
	resp := postXiaozhi(t, "set_music", url.Values{"source": {"youtube"}, "url": {" music.local:5005/ "}})
	if resp["status"] != "success" {
		t.Fatalf("set_music: %v", resp)
	}
	cfg, _ := loadXiaozhiCfg()
	if cfg.MusicSource != xiaozhiMusicYouTube || cfg.MusicServerURL != "https://music.local:5005" {
		t.Fatalf("saved %q %q", cfg.MusicSource, cfg.MusicServerURL)
	}

	postXiaozhi(t, "set_music", url.Values{"source": {"server"}, "url": {""}})
	cfg, _ = loadXiaozhiCfg()
	if cfg.MusicSource != xiaozhiMusicServer || cfg.MusicServerURL != xiaozhiMusicDefaultURL {
		t.Fatalf("saved %q %q", cfg.MusicSource, cfg.MusicServerURL)
	}
}

func TestXiaozhiSetMusicRejectsUnknownSource(t *testing.T) {
	withTempXiaozhiPaths(t)
	if resp := postXiaozhi(t, "set_music", url.Values{"source": {"spotify"}}); resp["status"] == "success" {
		t.Fatal("unknown source accepted")
	}
}

func TestXiaozhiOtherSavesKeepMusic(t *testing.T) {
	withTempXiaozhiPaths(t)
	postXiaozhi(t, "set_music", url.Values{"source": {"youtube"}, "url": {"https://m.example"}})
	postXiaozhi(t, "set_listen_mode", url.Values{"mode": {"vad"}})
	cfg, _ := loadXiaozhiCfg()
	if cfg.MusicSource != xiaozhiMusicYouTube || cfg.MusicServerURL != "https://m.example" {
		t.Fatalf("listen-mode save dropped music: %q %q", cfg.MusicSource, cfg.MusicServerURL)
	}
}

func TestMusicServerProbe(t *testing.T) {
	streamCode := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/search":
			_, _ = io.WriteString(w, `{"success":true,"data":[{"id":"abc","title":"Lạc Trôi"}]}`)
		case "/api/stream/mp3":
			w.WriteHeader(streamCode)
			if streamCode != http.StatusOK {
				_, _ = io.WriteString(w, `{"error":"No supported JavaScript runtime"}`)
				return
			}
			_, _ = w.Write([]byte{0xff, 0xfb, 0x90, 0x00})
		}
	}))
	defer srv.Close()

	res := testMusicServer(srv.URL, "lac troi")
	if !res.SearchOK || !res.StreamOK || res.Title != "Lạc Trôi" || res.Error != "" {
		t.Fatalf("ok server: %+v", res)
	}
	streamCode = http.StatusInternalServerError
	res = testMusicServer(srv.URL, "lac troi")
	if !res.SearchOK || res.StreamOK || res.Error != "stream HTTP 500: No supported JavaScript runtime" {
		t.Fatalf("stream 500: %+v", res)
	}
}
