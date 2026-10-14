package mods

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
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

func TestXiaozhiSetFaceGreetInterval(t *testing.T) {
	withTempXiaozhiPaths(t)
	postXiaozhi(t, "set_face_greet", url.Values{"enabled": {"true"}})
	resp := postXiaozhi(t, "set_face_greet", url.Values{"every_min": {"15"}})
	cfg, _ := loadXiaozhiCfg()
	if resp["status"] != "success" || !cfg.FaceGreet || cfg.FaceGreetEveryMin == nil || *cfg.FaceGreetEveryMin != 15 {
		t.Fatalf("resp %v cfg face_greet=%v every=%v", resp, cfg.FaceGreet, cfg.FaceGreetEveryMin)
	}
	for _, bad := range []string{"-3", "abc", "241"} {
		if r := postXiaozhi(t, "set_face_greet", url.Values{"every_min": {bad}}); r["status"] == "success" {
			t.Fatalf("every_min=%q accepted", bad)
		}
	}
	if cfg, _ := loadXiaozhiCfg(); *cfg.FaceGreetEveryMin != 15 {
		t.Fatalf("rejected values changed the interval to %d", *cfg.FaceGreetEveryMin)
	}
	postXiaozhi(t, "set_face_greet", url.Values{"every_min": {"0"}})
	data, _ := os.ReadFile(xiaozhiConfigPath)
	if cfg, _ := loadXiaozhiCfg(); cfg.FaceGreetEveryMin == nil || *cfg.FaceGreetEveryMin != 0 ||
		!strings.Contains(string(data), `"face_greet_every_min"`) {
		t.Fatalf("no-wait (0) not saved: %s", data)
	}
}

func TestXiaozhiSetTTSGain(t *testing.T) {
	withTempXiaozhiPaths(t)
	if r := postXiaozhi(t, "set_tts_gain", url.Values{"pct": {"160"}}); r["status"] != "success" {
		t.Fatalf("resp %v", r)
	}
	if cfg, _ := loadXiaozhiCfg(); cfg.TTSGainPct != 160 {
		t.Fatalf("tts_gain_pct = %d", cfg.TTSGainPct)
	}
	for _, bad := range []string{"", "99", "1001", "x"} {
		if r := postXiaozhi(t, "set_tts_gain", url.Values{"pct": {bad}}); r["status"] == "success" {
			t.Fatalf("pct=%q accepted", bad)
		}
	}
	if cfg, _ := loadXiaozhiCfg(); cfg.TTSGainPct != 160 {
		t.Fatalf("rejected value changed gain to %d", cfg.TTSGainPct)
	}
}
