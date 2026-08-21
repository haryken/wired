package mods

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/os-vector/wired/vars"
)

// genericBoardMod wraps boardGameHTTP for extra minigames.
type genericBoardMod struct {
	vars.Modification
	api  boardGameHTTP
	name string
	desc string
}

func (m *genericBoardMod) Name() string        { return m.name }
func (m *genericBoardMod) Description() string { return m.desc }
func (m *genericBoardMod) Load() error         { return nil }
func (m *genericBoardMod) HTTP(w http.ResponseWriter, r *http.Request) {
	m.api.Serve(w, r)
}

func newGenericBoardMod(apiName, exitID, desc string,
	summary func() string,
	snapshot func() map[string]interface{},
	reset func() map[string]interface{},
	setDiff func(string) string,
	getDiff func() string,
	play func(string) (map[string]interface{}, error),
	legal func() []string,
	newSpeak func() (string, string),
) *genericBoardMod {
	m := &genericBoardMod{name: apiName, desc: desc}
	m.api = boardGameHTTP{
		name: apiName, exitDefault: exitID,
		summary: summary, snapshot: snapshot, reset: reset,
		setDifficulty: setDiff, getDifficulty: getDiff,
		playUCI: play, legalUCIs: legal, newGameSpeak: newSpeak,
	}
	return m
}

type miniCommon struct {
	mu          sync.Mutex
	status      string // playing | win | lose | draw
	winner      string // human | bot | ""
	message     string
	lastMove    string
	history     []string
	difficulty  string
	humanTurn   bool
	botThinking bool
	thinkGen    int
	moves       int
}

func (c *miniCommon) baseSnap(game, uiMode string, extra map[string]interface{}) map[string]interface{} {
	turn := "bot"
	if c.humanTurn {
		turn = "human"
	}
	out := map[string]interface{}{
		"turn": turn, "status": c.status, "winner": c.winner,
		"lastMove": c.lastMove, "history": append([]string{}, c.history...),
		"message": c.message, "difficulty": normalizeGameDifficulty(c.difficulty),
		"botThinking": c.botThinking, "game": game, "uiMode": uiMode,
		"placeMode": uiMode == "place" || uiMode == "tictactoe" || uiMode == "mines" || uiMode == "memory" || uiMode == "battleship",
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func viOrEN(vi, en string) (string, string) {
	if chessPreferVIText() {
		return vi, vi
	}
	return en, en
}

func miniSqName(file, rank int) string {
	return string(rune('a'+file)) + fmt.Sprintf("%d", rank)
}

func miniParseSq(s string) (file, rank int, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 || s[0] < 'a' || s[0] > 'z' {
		return 0, 0, false
	}
	file = int(s[0] - 'a')
	rank = 0
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, 0, false
		}
		rank = rank*10 + int(s[i]-'0')
	}
	return file, rank, true
}
