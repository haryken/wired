package mods

import (
	"net/http"

	"github.com/os-vector/wired/vars"
)

// Connect4 is classic 7×6 Connect Four on web Games.
type Connect4 struct {
	vars.Modification
	api boardGameHTTP
}

func NewConnect4() *Connect4 {
	m := &Connect4{}
	m.api = boardGameHTTP{
		name:          "Connect4",
		exitDefault:   "connect4",
		summary:       func() string { return getConnect4().summaryText() },
		snapshot:      func() map[string]interface{} { return getConnect4().snapshot() },
		reset:         func() map[string]interface{} { return getConnect4().reset() },
		setDifficulty: func(s string) string { return getConnect4().setDifficulty(s) },
		getDifficulty: func() string { return getConnect4().getDifficulty() },
		playUCI:       func(u string) (map[string]interface{}, error) { return getConnect4().playUCI(u) },
		legalUCIs:     func() []string { return getConnect4().legalUCIs() },
		newGameSpeak: func() (string, string) {
			if chessPreferVIText() {
				return "Ván Connect Four mới. Bạn thả trước. Đến lượt bạn.", "Connect Four. Ván mới."
			}
			return "New Connect Four. You drop first. Your move.", "Connect Four. Ván mới."
		},
	}
	return m
}

func (m *Connect4) Name() string { return "Connect4" }

func (m *Connect4) Description() string {
	return "Connect Four web game — Vector comments; Xiaozhi MCP"
}

func (m *Connect4) Load() error {
	_ = getConnect4()
	return nil
}

func (m *Connect4) HTTP(w http.ResponseWriter, r *http.Request) {
	m.api.Serve(w, r)
}
