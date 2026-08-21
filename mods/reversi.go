package mods

import (
	"net/http"

	"github.com/os-vector/wired/vars"
)

// Reversi is Othello 8×8 on the web Games tab.
type Reversi struct {
	vars.Modification
	api boardGameHTTP
}

func NewReversi() *Reversi {
	m := &Reversi{}
	m.api = boardGameHTTP{
		name:          "Reversi",
		exitDefault:   "reversi",
		summary:       func() string { return getReversi().summaryText() },
		snapshot:      func() map[string]interface{} { return getReversi().snapshot() },
		reset:         func() map[string]interface{} { return getReversi().reset() },
		setDifficulty: func(s string) string { return getReversi().setDifficulty(s) },
		getDifficulty: func() string { return getReversi().getDifficulty() },
		playUCI:       func(u string) (map[string]interface{}, error) { return getReversi().playUCI(u) },
		legalUCIs:     func() []string { return getReversi().legalUCIs() },
		newGameSpeak: func() (string, string) {
			if chessPreferVIText() {
				return "Ván Reversi mới. Bạn cầm đen. Đến lượt bạn.", "Reversi. Ván mới."
			}
			return "New Reversi. You are black. Your move.", "Reversi. Ván mới."
		},
	}
	return m
}

func (m *Reversi) Name() string { return "Reversi" }

func (m *Reversi) Description() string {
	return "Reversi/Othello web game — Vector comments; Xiaozhi MCP"
}

func (m *Reversi) Load() error {
	_ = getReversi()
	return nil
}

func (m *Reversi) HTTP(w http.ResponseWriter, r *http.Request) {
	m.api.Serve(w, r)
}
