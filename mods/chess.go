package mods

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/os-vector/wired/vars"
)

// Chess is a web-board chess game (human White vs bot Black) for :80 UI + Xiaozhi MCP.
type Chess struct {
	vars.Modification
}

func NewChess() *Chess {
	return &Chess{}
}

func (m *Chess) Name() string { return "Chess" }

func (m *Chess) Description() string {
	return "Chess on web :80 — play vs bot; auto Vector comments; Xiaozhi MCP reads board"
}

func (m *Chess) Load() error {
	_ = getChess()
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (m *Chess) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	g := getChess()
	switch path {
	case "state", "summary":
		if path == "summary" {
			writeJSON(w, 200, map[string]string{"text": g.summaryText()})
			return
		}
		st := g.snapshot()
		st["comment"] = chessCommentEnabled()
		st["commentMode"] = getChessCommentMode()
		st["xiaozhiAvailable"] = chessXiaozhiAvailable()
		st["googleVIAvailable"] = chessGoogleVIAvailable()
		st["difficulty"] = g.getDifficulty()
		writeJSON(w, 200, st)
	case "comment":
		// GET → current; POST/GET ?enable=0|1 → set
		if v := r.FormValue("enable"); v != "" {
			on := v == "1" || strings.EqualFold(v, "true") || v == "on"
			setChessCommentEnabled(on)
		}
		writeJSON(w, 200, map[string]interface{}{
			"comment":           chessCommentEnabled(),
			"commentMode":       getChessCommentMode(),
			"xiaozhiAvailable":  chessXiaozhiAvailable(),
			"googleVIAvailable": chessGoogleVIAvailable(),
			"difficulty":        g.getDifficulty(),
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
			"difficulty":        g.getDifficulty(),
		})
	case "difficulty":
		if v := r.FormValue("level"); v != "" {
			g.setDifficulty(v)
		}
		writeJSON(w, 200, map[string]interface{}{
			"difficulty": g.getDifficulty(),
			"label":      difficultyLabelVI(g.getDifficulty()),
		})
	case "new":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			vars.HTTPError(w, r, "POST required")
			return
		}
		// Optional level on new game.
		if v := r.FormValue("level"); v != "" {
			g.setDifficulty(v)
		}
		st := g.reset()
		st["comment"] = chessCommentEnabled()
		st["commentMode"] = getChessCommentMode()
		st["xiaozhiAvailable"] = chessXiaozhiAvailable()
		st["googleVIAvailable"] = chessGoogleVIAvailable()
		st["difficulty"] = g.getDifficulty()
		writeJSON(w, 200, st)
		if getChessCommentMode() == chessModeGoogleVI {
			queueChessSpeak("Ván mới. Bạn cầm trắng. Đến lượt bạn.", getChess().summaryText())
		} else {
			queueChessSpeak("New game. You are white. Your move.", getChess().summaryText())
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
		st, err := g.playUCI(uci)
		if err != nil {
			writeJSON(w, 400, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		st["comment"] = chessCommentEnabled()
		st["commentMode"] = getChessCommentMode()
		st["xiaozhiAvailable"] = chessXiaozhiAvailable()
		st["googleVIAvailable"] = chessGoogleVIAvailable()
		st["difficulty"] = g.getDifficulty()
		writeJSON(w, 200, st)
		// TTS is queued inside playUCI / runBotThink (per-move, not after both).
	case "legal":
		writeJSON(w, 200, map[string]interface{}{"moves": g.legalUCIs()})
	case "exit":
		// Leave game UI: always SayText announce (ignore comment mode / toggle).
		name := strings.TrimSpace(r.FormValue("game"))
		if name == "" {
			name = "chess"
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
