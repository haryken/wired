package mods

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/os-vector/wired/vars"
)

// WebLocale stores the web UI language preference on the robot so every browser
// session shares the same choice.
type WebLocale struct {
	vars.Modification
}

const webLocaleName = "WebLocale"

var webLocalePath = "/data/data/com.anki.victor/persistent/wired/ui_locale.json"

type webLocaleCfg struct {
	Lang string `json:"lang"`
}

func NewWebLocale() *WebLocale {
	return &WebLocale{}
}

func (m *WebLocale) Name() string { return webLocaleName }

func (m *WebLocale) Description() string {
	return "Web UI language preference (en/vi/it/zh) — persisted on robot"
}

func (m *WebLocale) Load() error {
	_ = loadWebLocale()
	return nil
}

func normalizeUILang(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "en", "en-us", "en-gb", "english":
		return "en"
	case "it", "it-it", "italian", "italiano":
		return "it"
	case "zh", "zh-cn", "zh-hans", "cn", "chinese", "中文":
		return "zh"
	case "vi", "vi-vn", "vietnamese", "vn":
		return "vi"
	default:
		return "vi"
	}
}

func loadWebLocale() webLocaleCfg {
	cfg := webLocaleCfg{Lang: "vi"}
	b, err := os.ReadFile(webLocalePath)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	cfg.Lang = normalizeUILang(cfg.Lang)
	return cfg
}

func saveWebLocale(cfg webLocaleCfg) error {
	cfg.Lang = normalizeUILang(cfg.Lang)
	if err := os.MkdirAll(filepath.Dir(webLocalePath), 0777); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(webLocalePath, data, 0644)
}

func (m *WebLocale) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	switch path {
	case "get":
		cfg := loadWebLocale()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"lang":   cfg.Lang,
		})
	case "set":
		lang := r.FormValue("lang")
		if lang == "" {
			vars.HTTPError(w, r, "lang required")
			return
		}
		cfg := webLocaleCfg{Lang: normalizeUILang(lang)}
		if err := saveWebLocale(cfg); err != nil {
			vars.HTTPError(w, r, "save failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"lang":   cfg.Lang,
		})
	default:
		vars.HTTPError(w, r, "unknown endpoint")
	}
}
