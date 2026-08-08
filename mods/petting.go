package mods

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/os-vector/wired/vars"
)

// PettingEnabledFile gates BehaviorReactToTouchPetting (purr / pet reaction).
// TouchEnabledFile gates TouchSensorComponent (all backpack-touch features).
// Missing files = enabled (stock behavior).
var PettingEnabledFile = filepath.Join(vars.GetModDir("Petting"), "enabled")
var TouchEnabledFile = filepath.Join(vars.GetModDir("Petting"), "touch_enabled")

type Petting struct {
	vars.Modification
}

func NewPetting() *Petting {
	return &Petting{}
}

func (m *Petting) Name() string {
	return "Petting"
}

func (m *Petting) Description() string {
	return "Enable/disable backpack petting and optionally the whole touch sensor (false-touch workaround)."
}

func normalizeBoolFlag(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "false", "0", "off", "no", "disabled":
		return "false"
	default:
		return "true"
	}
}

func readBoolFile(path string) string {
	f, err := vars.ReadFile(path)
	if err != nil || strings.TrimSpace(f) == "" {
		return "true"
	}
	return normalizeBoolFlag(f)
}

func (m *Petting) HTTP(w http.ResponseWriter, r *http.Request) {
	if vars.IsEndpoint(r, "get") {
		out, _ := json.Marshal(map[string]string{
			"petting": readBoolFile(PettingEnabledFile),
			"touch":   readBoolFile(TouchEnabledFile),
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}
	if vars.IsEndpoint(r, "set_touch") {
		enabled := normalizeBoolFlag(r.FormValue("enabled"))
		if err := vars.SaveFile(enabled+"\n", TouchEnabledFile); err != nil {
			vars.HTTPError(w, r, "save touch: "+err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
		return
	}
	if vars.IsEndpoint(r, "set") {
		enabled := normalizeBoolFlag(r.FormValue("enabled"))
		if err := vars.SaveFile(enabled+"\n", PettingEnabledFile); err != nil {
			vars.HTTPError(w, r, "save: "+err.Error())
			return
		}
		vars.HTTPSuccess(w, r)
		return
	}
	vars.HTTPError(w, r, "unknown endpoint")
}

func (m *Petting) Load() error {
	if c, err := vars.ReadFile(PettingEnabledFile); err != nil || strings.TrimSpace(c) == "" {
		_ = vars.SaveFile("true\n", PettingEnabledFile)
	}
	if c, err := vars.ReadFile(TouchEnabledFile); err != nil || strings.TrimSpace(c) == "" {
		_ = vars.SaveFile("true\n", TouchEnabledFile)
	}
	return nil
}
