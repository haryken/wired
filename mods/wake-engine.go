package mods

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/os-vector/wired/vars"
)

// WakeEngineLocation selects vic-anim wake backend (picovoice | thf).
var WakeEngineLocation = "/data/data/com.anki.victor/persistent/wake_engine"

const (
	WakeEnginePicovoice = "picovoice"
	WakeEngineTHF       = "thf"
)

type WakeEngine struct {
	vars.Modification
}

func NewWakeEngine() *WakeEngine {
	return &WakeEngine{}
}

func (modu *WakeEngine) Name() string {
	return "WakeEngine"
}

func (modu *WakeEngine) Description() string {
	return "Switch Vector wake word engine: Picovoice or Sensory THF (stock Hey Vector)."
}

func normalizeWakeEngine(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case WakeEnginePicovoice, "pv", "porcupine", "custom":
		return WakeEnginePicovoice
	default:
		// Empty / missing / thf / sensory → WireOS OTA default THF
		return WakeEngineTHF
	}
}

func (modu *WakeEngine) HTTP(w http.ResponseWriter, r *http.Request) {
	if vars.IsEndpoint(r, "get") {
		f, err := vars.ReadFile(WakeEngineLocation)
		if err != nil || strings.TrimSpace(f) == "" {
			f = WakeEngineTHF
		}
		w.Write([]byte(normalizeWakeEngine(f)))
		return
	}
	if vars.IsEndpoint(r, "set") {
		engine := normalizeWakeEngine(r.FormValue("engine"))
		if err := os.MkdirAll(filepath.Dir(WakeEngineLocation), 0770); err != nil {
			vars.HTTPError(w, r, "mkdir: "+err.Error())
			return
		}
		vars.SaveFile(engine+"\n", WakeEngineLocation)
		vars.HTTPSuccess(w, r)
		return
	}
	vars.HTTPError(w, r, "unknown endpoint")
}

func (modu *WakeEngine) Load() error {
	// Seed OTA default on first boot so file exists for anim + UI.
	// SaveFile → SetAnkiPerms so this does not leave persistent/ as root:root
	// (that blocks vic-switchboard sessions after CLEAR OUT SOUL → fault 913).
	if _, err := os.Stat(WakeEngineLocation); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(WakeEngineLocation), 0770)
		vars.SaveFile(WakeEngineTHF+"\n", WakeEngineLocation)
	}
	return nil
}
