package mods

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func withTempXiaozhiPaths(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	oldCfg, oldFlag := xiaozhiConfigPath, xiaozhiListenVADFlagPath
	xiaozhiConfigPath = filepath.Join(dir, "persistent", "xiaozhi.json")
	xiaozhiListenVADFlagPath = filepath.Join(dir, "run", "xiaozhi-listen-vad")
	t.Cleanup(func() {
		xiaozhiConfigPath, xiaozhiListenVADFlagPath = oldCfg, oldFlag
	})
	return dir
}

func setListenMode(t *testing.T, mode string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_listen_mode?mode="+mode, nil)
	w := httptest.NewRecorder()
	NewXiaozhi().HTTP(w, req)
	return w
}

func flagExists() bool {
	_, err := os.Stat(xiaozhiListenVADFlagPath)
	return err == nil
}

func TestXiaozhiSetListenModeTogglesFlag(t *testing.T) {
	withTempXiaozhiPaths(t)

	if w := setListenMode(t, "vad"); w.Code != 200 {
		t.Fatalf("vad: HTTP %d %s", w.Code, w.Body.String())
	}
	cfg, err := loadXiaozhiCfg()
	if err != nil || cfg.ListenMode != xiaozhiListenVAD {
		t.Fatalf("listen_mode %q err %v", cfg.ListenMode, err)
	}
	if !flagExists() {
		t.Fatal("vad mode did not create flag")
	}

	setListenMode(t, "fixed")
	cfg, _ = loadXiaozhiCfg()
	if cfg.ListenMode != xiaozhiListenFixed || flagExists() {
		t.Fatalf("fixed: listen_mode %q flag %v", cfg.ListenMode, flagExists())
	}
}

func TestXiaozhiSetListenModeRejectsUnknown(t *testing.T) {
	withTempXiaozhiPaths(t)
	w := setListenMode(t, "always")
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] == "success" {
		t.Fatal("unknown mode accepted")
	}
	if _, err := os.Stat(xiaozhiConfigPath); err == nil {
		t.Fatal("config written for rejected mode")
	}
}

func TestXiaozhiDisabledClearsFlag(t *testing.T) {
	withTempXiaozhiPaths(t)
	setListenMode(t, "vad")
	cfg, _ := loadXiaozhiCfg()
	cfg.Enabled = false
	if err := saveXiaozhiCfg(cfg); err != nil {
		t.Fatal(err)
	}
	if flagExists() {
		t.Fatal("flag kept while Xiaozhi disabled (Vosk path must keep stock mic window)")
	}
}

func TestXiaozhiSaveKeepsListenMode(t *testing.T) {
	withTempXiaozhiPaths(t)
	setListenMode(t, "vad")
	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/save?conversation_mode=continuous&identity_mode=vi_pool", nil)
	NewXiaozhi().HTTP(httptest.NewRecorder(), req)
	cfg, _ := loadXiaozhiCfg()
	if cfg.ListenMode != xiaozhiListenVAD || !flagExists() {
		t.Fatalf("Xiaozhi save dropped listen_mode: %q flag %v", cfg.ListenMode, flagExists())
	}
}
