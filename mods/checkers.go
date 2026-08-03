package mods

import (
	"net/http"

	"github.com/os-vector/wired/vars"
)

// Checkers is English draughts on the web Games tab.
type Checkers struct {
	vars.Modification
	api boardGameHTTP
}

func NewCheckers() *Checkers {
	m := &Checkers{}
	m.api = boardGameHTTP{
		name:          "Checkers",
		exitDefault:   "checkers",
		summary:       func() string { return getCheckers().summaryText() },
		snapshot:      func() map[string]interface{} { return getCheckers().snapshot() },
		reset:         func() map[string]interface{} { return getCheckers().reset() },
		setDifficulty: func(s string) string { return getCheckers().setDifficulty(s) },
		getDifficulty: func() string { return getCheckers().getDifficulty() },
		playUCI:       func(u string) (map[string]interface{}, error) { return getCheckers().playUCI(u) },
		legalUCIs:     func() []string { return getCheckers().legalUCIs() },
		newGameSpeak: func() (string, string) {
			if getChessCommentMode() == chessModeGoogleVI {
				return "Ván cờ đam mới. Bạn cầm trắng. Đến lượt bạn.", "Cờ đam. Ván mới."
			}
			return "New checkers game. You are white. Your move.", "Cờ đam. Ván mới."
		},
	}
	return m
}

func (m *Checkers) Name() string { return "Checkers" }

func (m *Checkers) Description() string {
	return "Checkers/draughts web game — Vector comments; Xiaozhi MCP"
}

func (m *Checkers) Load() error {
	_ = getCheckers()
	return nil
}

func (m *Checkers) HTTP(w http.ResponseWriter, r *http.Request) {
	m.api.Serve(w, r)
}
