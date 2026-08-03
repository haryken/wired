package mods

import (
	"net/http"

	"github.com/os-vector/wired/vars"
)

// Go9 is micro Go 9×9 on the web Games tab.
type Go9 struct {
	vars.Modification
	api boardGameHTTP
}

func NewGo9() *Go9 {
	m := &Go9{}
	m.api = boardGameHTTP{
		name:          "Go9",
		exitDefault:   "go9",
		summary:       func() string { return getGo9().summaryText() },
		snapshot:      func() map[string]interface{} { return getGo9().snapshot() },
		reset:         func() map[string]interface{} { return getGo9().reset() },
		setDifficulty: func(s string) string { return getGo9().setDifficulty(s) },
		getDifficulty: func() string { return getGo9().getDifficulty() },
		playUCI:       func(u string) (map[string]interface{}, error) { return getGo9().playUCI(u) },
		legalUCIs:     func() []string { return getGo9().legalUCIs() },
		newGameSpeak: func() (string, string) {
			if getChessCommentMode() == chessModeGoogleVI {
				return "Ván cờ vây 9×9 mới. Bạn cầm đen. Đến lượt bạn.", "Cờ vây 9×9. Ván mới."
			}
			return "New 9 by 9 go game. You are black. Your move.", "Cờ vây 9×9. Ván mới."
		},
	}
	return m
}

func (m *Go9) Name() string { return "Go9" }

func (m *Go9) Description() string {
	return "Go 9×9 web game — Vector comments; Xiaozhi MCP"
}

func (m *Go9) Load() error {
	_ = getGo9()
	return nil
}

func (m *Go9) HTTP(w http.ResponseWriter, r *http.Request) {
	m.api.Serve(w, r)
}
