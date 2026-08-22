package mods

import (
	"net/http"

	"github.com/os-vector/wired/vars"
)

// Caro is freestyle gomoku with cấm 2 đầu for X (human).
type Caro struct {
	vars.Modification
	api boardGameHTTP
}

func NewCaro() *Caro {
	m := &Caro{}
	m.api = boardGameHTTP{
		name:          "Caro",
		exitDefault:   "caro",
		summary:       func() string { return getCaro().summaryText() },
		snapshot:      func() map[string]interface{} { return getCaro().snapshot() },
		reset:         func() map[string]interface{} { return getCaro().reset() },
		setDifficulty: func(s string) string { return getCaro().setDifficulty(s) },
		getDifficulty: func() string { return getCaro().getDifficulty() },
		playUCI:       func(u string) (map[string]interface{}, error) { return getCaro().playUCI(u) },
		legalUCIs:     func() []string { return getCaro().legalUCIs() },
		newGameSpeak: func() (string, string) {
			return speakNew(
				"New caro game. You are X. Freestyle with double-open bans. Your move.",
				"Ván caro mới. Bạn cầm X. Freestyle, cấm 2 đầu. Đến lượt bạn.",
			)
		},
	}
	return m
}

func (m *Caro) Name() string { return "Caro" }

func (m *Caro) Description() string {
	return "Caro/gomoku freestyle + cấm 2 đầu on web :80; Vector comments; Xiaozhi MCP"
}

func (m *Caro) Load() error {
	_ = getCaro()
	return nil
}

func (m *Caro) HTTP(w http.ResponseWriter, r *http.Request) {
	m.api.Serve(w, r)
}
