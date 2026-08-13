package mods

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/os-vector/wired/vars"
)

// FakeCamera writes /data/wired/mods/FakeCamera/mode for vic-engine.
// Missing file = auto (fake YUV only when hardware CameraGetFrame fails).
type FakeCamera struct {
	vars.Modification
}

func NewFakeCamera() *FakeCamera {
	return &FakeCamera{}
}

func (m *FakeCamera) Name() string { return "FakeCamera" }

func (m *FakeCamera) Description() string {
	return "Empty-room fake camera for broken-camera robots (auto/on/off)"
}

func (m *FakeCamera) Load() error { return nil }

func fakeCameraModePath() string {
	return filepath.Join(vars.GetModDir("FakeCamera"), "mode")
}

func normalizeFakeCameraMode(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "on", "force", "true", "1":
		return "on"
	case "off", "never", "false", "0":
		return "off"
	default:
		return "auto"
	}
}

func readFakeCameraMode() string {
	raw, err := vars.ReadFile(fakeCameraModePath())
	if err != nil {
		return "auto"
	}
	return normalizeFakeCameraMode(raw)
}

func (m *FakeCamera) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	switch path {
	case "get_mode":
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"mode": readFakeCameraMode(),
		})
	case "set_mode":
		_ = r.ParseForm()
		mode := normalizeFakeCameraMode(r.FormValue("mode"))
		if err := vars.SaveFile(mode+"\n", fakeCameraModePath()); err != nil {
			vars.HTTPError(w, r, err.Error())
			return
		}
		// Keep world-readable so engine user can poll without wired restart.
		_ = os.Chmod(fakeCameraModePath(), 0644)
		vars.HTTPSuccess(w, r)
	default:
		vars.HTTPError(w, r, fmt.Sprintf("404 not found"))
	}
}
