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
	case WakeEngineTHF, "sensory", "hey_vector_thf", "heyvector", "stock":
		return WakeEngineTHF
	default:
		return WakeEnginePicovoice
	}
}

func (modu *WakeEngine) HTTP(w http.ResponseWriter, r *http.Request) {
	if vars.IsEndpoint(r, "get") {
		f, err := vars.ReadFile(WakeEngineLocation)
		if err != nil || strings.TrimSpace(f) == "" {
			f = WakeEnginePicovoice
		}
		w.Write([]byte(normalizeWakeEngine(f)))
		return
	}
	if vars.IsEndpoint(r, "set") {
		engine := normalizeWakeEngine(r.FormValue("engine"))
		if err := os.MkdirAll(filepath.Dir(WakeEngineLocation), 0777); err != nil {
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
	return nil
}
