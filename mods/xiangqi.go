package mods

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/os-vector/wired/vars"
)

// Xiangqi is a web-board Chinese chess game (human Red vs bot Black) for
// :80 UI + Xiaozhi MCP, mirroring the Chess mod.
type Xiangqi struct {
	vars.Modification
}

func NewXiangqi() *Xiangqi {
	return &Xiangqi{}
}

func (m *Xiangqi) Name() string { return "Xiangqi" }

func (m *Xiangqi) Description() string {
	return "Xiangqi (cờ tướng) on web :80 — play vs bot; auto Vector comments; Xiaozhi MCP reads board"
}

func (m *Xiangqi) Load() error {
	_ = getXiangqi()
	return nil
}

func (m *Xiangqi) HTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/mods/"+m.Name()) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/mods/"+m.Name()+"/")
	g := getXiangqi()
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
		st["googleTTSLang"] = chessGoogleTTSLang()
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
			"googleTTSLang":     chessGoogleTTSLang(),
			"difficulty":        g.getDifficulty(),
		})
	case "comment_mode":
		if v := r.FormValue("mode"); v != "" {
			setChessCommentMode(v)
		}
		if v := r.FormValue("lang"); v != "" {
			setChessGoogleTTSLang(v)
		}
		writeJSON(w, 200, map[string]interface{}{
			"commentMode":       getChessCommentMode(),
			"xiaozhiAvailable":  chessXiaozhiAvailable(),
			"googleVIAvailable": chessGoogleVIAvailable(),
			"googleTTSLang":     chessGoogleTTSLang(),
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
		if v := r.FormValue("level"); v != "" {
			g.setDifficulty(v)
		}
		st := g.reset()
		st["comment"] = chessCommentEnabled()
		st["commentMode"] = getChessCommentMode()
		st["xiaozhiAvailable"] = chessXiaozhiAvailable()
		st["googleVIAvailable"] = chessGoogleVIAvailable()
		st["googleTTSLang"] = chessGoogleTTSLang()
		st["difficulty"] = g.getDifficulty()
		writeJSON(w, 200, st)
		queueChessSpeak(xiangqiNewGameSpeak(), getXiangqi().summaryText())
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
		st["googleTTSLang"] = chessGoogleTTSLang()
		st["difficulty"] = g.getDifficulty()
		writeJSON(w, 200, st)
		// TTS is queued inside playUCI / runBotThinkXQ (per-move).
	case "legal":
		writeJSON(w, 200, map[string]interface{}{"moves": g.legalUCIs()})
	case "exit":
		// Leave game UI: always SayText announce (ignore comment mode / toggle).
		name := strings.TrimSpace(r.FormValue("game"))
		if name == "" {
			name = "xiangqi"
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
