package mods

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/os-vector/wired/vars"
)

// boardGameHTTP is the shared /api/mods/{Name}/ surface for web games + Xiaozhi.
type boardGameHTTP struct {
	name          string // API path segment, e.g. "Caro"
	exitDefault   string // speak id, e.g. "caro"
	summary       func() string
	snapshot      func() map[string]interface{}
	reset         func() map[string]interface{}
	setDifficulty func(string) string
	getDifficulty func() string
	playUCI       func(string) (map[string]interface{}, error)
	legalUCIs     func() []string
	newGameSpeak  func() (say string, summaryHint string)
}

func attachGameCaps(st map[string]interface{}, difficulty string) {
	if st == nil {
		return
	}
	st["comment"] = chessCommentEnabled()
	st["commentMode"] = getChessCommentMode()
	st["xiaozhiAvailable"] = chessXiaozhiAvailable()
	st["googleVIAvailable"] = chessGoogleVIAvailable()
	st["difficulty"] = difficulty
}

func (h *boardGameHTTP) Serve(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+h.name) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+h.name+"/")
	switch path {
	case "state", "summary":
		if path == "summary" {
			writeJSON(w, 200, map[string]string{"text": h.summary()})
			return
		}
		st := h.snapshot()
		attachGameCaps(st, h.getDifficulty())
		writeJSON(w, 200, st)
	case "comment":
		if v := r.FormValue("enable"); v != "" {
			on := v == "1" || strings.EqualFold(v, "true") || v == "on"
			setChessCommentEnabled(on)
		}
		writeJSON(w, 200, map[string]interface{}{
			"comment":           chessCommentEnabled(),
			"commentMode":       getChessCommentMode(),
			"xiaozhiAvailable":  chessXiaozhiAvailable(),
			"googleVIAvailable": chessGoogleVIAvailable(),
			"difficulty":        h.getDifficulty(),
		})
	case "comment_mode":
		if v := r.FormValue("mode"); v != "" {
			setChessCommentMode(v)
		}
		writeJSON(w, 200, map[string]interface{}{
			"commentMode":       getChessCommentMode(),
			"xiaozhiAvailable":  chessXiaozhiAvailable(),
			"googleVIAvailable": chessGoogleVIAvailable(),
			"comment":           chessCommentEnabled(),
			"difficulty":        h.getDifficulty(),
		})
	case "difficulty":
		if v := r.FormValue("level"); v != "" {
			h.setDifficulty(v)
		}
		writeJSON(w, 200, map[string]interface{}{
			"difficulty": h.getDifficulty(),
			"label":      difficultyLabelVI(h.getDifficulty()),
		})
	case "new":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			vars.HTTPError(w, r, "POST required")
			return
		}
		if v := r.FormValue("level"); v != "" {
			h.setDifficulty(v)
		}
		st := h.reset()
		attachGameCaps(st, h.getDifficulty())
		writeJSON(w, 200, st)
		if h.newGameSpeak != nil {
			say, sum := h.newGameSpeak()
			if say != "" {
				queueChessSpeak(say, sum)
			}
		}
	case "move":
		if r.Method != http.MethodPost {
			vars.HTTPError(w, r, "POST required")
			return
		}
		uci := r.FormValue("uci")
		if uci == "" {
			var body struct {
				UCI string `json:"uci"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			uci = body.UCI
		}
		st, err := h.playUCI(uci)
		if err != nil {
			writeJSON(w, 400, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		attachGameCaps(st, h.getDifficulty())
		writeJSON(w, 200, st)
	case "legal":
		writeJSON(w, 200, map[string]interface{}{"moves": h.legalUCIs()})
	case "exit":
		name := strings.TrimSpace(r.FormValue("game"))
		if name == "" {
			name = h.exitDefault
		}
		msg := "Exiting " + speakGameName(name) + "."
		queueChessSpeakSayTextAlways(msg)
		writeJSON(w, 200, map[string]interface{}{
			"status":  "ok",
			"message": msg,
			"game":    name,
		})
	default:
		vars.HTTPError(w, r, "not found")
	}
}
