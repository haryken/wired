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

func TestXiaozhiSetConversationMode(t *testing.T) {
	dir := withTempXiaozhiPaths(t)
	oldAmbient, oldMode, oldArm := xiaozhiAmbientFlagPath, xiaozhiOpenMicModeFlagPath, xiaozhiOpenMicArmPath
	xiaozhiAmbientFlagPath = filepath.Join(dir, "run", "xiaozhi-ambient")
	xiaozhiOpenMicModeFlagPath = filepath.Join(dir, "run", "xiaozhi-openmic-mode")
	xiaozhiOpenMicArmPath = filepath.Join(dir, "run", "xiaozhi-openmic")
	t.Cleanup(func() {
		xiaozhiAmbientFlagPath, xiaozhiOpenMicModeFlagPath, xiaozhiOpenMicArmPath = oldAmbient, oldMode, oldArm
	})
	if err := os.MkdirAll(filepath.Dir(xiaozhiAmbientFlagPath), 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xiaozhiAmbientFlagPath, []byte("1"), 0664); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_conversation_mode?mode=open", nil)
	w := httptest.NewRecorder()
	NewXiaozhi().HTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("open: HTTP %d %s", w.Code, w.Body.String())
	}
	cfg, err := loadXiaozhiCfg()
	if err != nil || cfg.ConversationMode != "open" {
		t.Fatalf("conversation_mode %q err %v", cfg.ConversationMode, err)
	}
	if _, err := os.Stat(xiaozhiOpenMicModeFlagPath); err != nil {
		t.Fatal("open mode did not write the mode flag")
	}
	if _, err := os.Stat(xiaozhiAmbientFlagPath); err == nil {
		t.Fatal("ambient hold kept after leaving Kéo dài")
	}

	gate := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_conversation_mode?mode=gate&rms=1200", nil)
	gw := httptest.NewRecorder()
	NewXiaozhi().HTTP(gw, gate)
	if gw.Code != 200 {
		t.Fatalf("gate: HTTP %d %s", gw.Code, gw.Body.String())
	}
	cfg, err = loadXiaozhiCfg()
	if err != nil || cfg.ConversationMode != "gate" || cfg.GateRMS != 1200 {
		t.Fatalf("gate mode %q rms %d err %v", cfg.ConversationMode, cfg.GateRMS, err)
	}
	if _, err := os.Stat(xiaozhiOpenMicModeFlagPath); err != nil {
		t.Fatal("gate mode did not keep the open-mic flag")
	}
	low := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_conversation_mode?mode=gate&rms=1", nil)
	lw := httptest.NewRecorder()
	NewXiaozhi().HTTP(lw, low)
	cfg, err = loadXiaozhiCfg()
	if err != nil || cfg.GateRMS != 100 {
		t.Fatalf("rms clamp %d err %v", cfg.GateRMS, err)
	}

	bad := httptest.NewRequest("POST", "/api/mods/Xiaozhi/set_conversation_mode?mode=always", nil)
	bw := httptest.NewRecorder()
	NewXiaozhi().HTTP(bw, bad)
	var resp map[string]interface{}
	_ = json.Unmarshal(bw.Body.Bytes(), &resp)
	if resp["status"] == "success" {
		t.Fatal("unknown conversation mode accepted")
	}
}

func TestXiaozhiListenModeDefaultsToVAD(t *testing.T) {
	withTempXiaozhiPaths(t)
	syncXiaozhiListenVADFlag(XiaozhiCfg{Enabled: true})
	if !flagExists() {
		t.Fatal("empty listen_mode must default to end-of-speech (vad)")
	}
	syncXiaozhiListenVADFlag(XiaozhiCfg{Enabled: true, ListenMode: xiaozhiListenFixed})
	if flagExists() {
		t.Fatal("explicit fixed must keep the ~6s window")
	}
}
