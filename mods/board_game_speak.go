package mods

import (
	"fmt"
	"strings"
)

// Generic place/move comments for board games (Caro, Connect4, Reversi, …).

func buildPlaceCommentEN(game, youMove, botMove, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, "You played "+youMove+".")
	}
	if botMove != "" {
		parts = append(parts, "I played "+botMove+".")
	}
	switch status {
	case "win", "won", "checkmate":
		if winner == "bot" || winner == "O" {
			parts = append(parts, "I win.")
		} else if winner != "" {
			parts = append(parts, "You win!")
		} else {
			parts = append(parts, "Game over.")
		}
	case "draw", "stalemate":
		parts = append(parts, "Draw.")
	}
	if len(parts) == 0 {
		return "Your turn."
	}
	return strings.Join(parts, " ")
}

func buildPlaceCommentVI(game, youMove, botMove, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, "Bạn đánh "+youMove+".")
	}
	if botMove != "" {
		parts = append(parts, "Bot đánh "+botMove+".")
	}
	switch status {
	case "win", "won", "checkmate":
		if winner == "bot" || winner == "O" {
			parts = append(parts, "Bot thắng.")
		} else if winner != "" {
			parts = append(parts, "Bạn thắng!")
		} else {
			parts = append(parts, "Hết ván.")
		}
	case "draw", "stalemate":
		parts = append(parts, "Hoà.")
	}
	if len(parts) == 0 {
		return "Đến lượt bạn."
	}
	return strings.Join(parts, " ")
}

func buildPlaceSpoken(game, youMove, botMove, status, winner string) string {
	lang := gameSpeakLang()
	if lang == "en" {
		return buildPlaceCommentEN(game, youMove, botMove, status, winner)
	}
	if lang == "vi" {
		return buildPlaceCommentVI(game, youMove, botMove, status, winner)
	}
	return buildPlaceCommentForLang(youMove, botMove, status, winner)
}

func buildPlaceSpokenHumanOnly(game, youMove, status, winner string) string {
	// While bot still to move, don't claim win for bot side.
	return buildPlaceSpoken(game, youMove, "", status, winner)
}

func buildPlaceSpokenBotOnly(game, botMove, status, winner string) string {
	return buildPlaceSpoken(game, "", botMove, status, winner)
}

// buildXiaozhiGamePrompt routes conversation-mode comments to the right MCP summarize tool.
func buildXiaozhiGamePrompt(gameID, facts, summary string) string {
	gid := strings.ToLower(strings.TrimSpace(gameID))
	if gid == "" {
		// Infer from summary / facts markers.
		blob := strings.ToLower(summary + " " + facts)
		switch {
		case strings.Contains(blob, "cờ tướng") || strings.Contains(blob, "xiangqi"):
			gid = "xiangqi"
		case strings.Contains(blob, "cờ vua") || strings.Contains(blob, "chess") && !strings.Contains(blob, "chinese"):
			gid = "chess"
		case strings.Contains(blob, "cờ caro") || strings.Contains(blob, "gomoku") || strings.Contains(blob, "caro"):
			gid = "caro"
		case strings.Contains(blob, "connect four") || strings.Contains(blob, "connect4") || strings.Contains(blob, "cờ thả"):
			gid = "connect4"
		case strings.Contains(blob, "reversi") || strings.Contains(blob, "othello"):
			gid = "reversi"
		case strings.Contains(blob, "checkers") || strings.Contains(blob, "cờ đam") || strings.Contains(blob, "draughts"):
			gid = "checkers"
		case strings.Contains(blob, "cờ vây") || strings.Contains(blob, " go ") || strings.HasPrefix(blob, "go ") || strings.Contains(blob, "go9") || strings.Contains(blob, "weiqi"):
			gid = "go9"
		default:
			gid = "chess"
		}
	}

	tool := "self." + gid + ".summarize"
	// chess/xiangqi keep historical tool names
	if gid == "chess" {
		tool = "self.chess.summarize"
	}
	if gid == "xiangqi" {
		tool = "self.xiangqi.summarize"
	}

	label := map[string]string{
		"chess":    "cờ vua (chess)",
		"xiangqi":  "cờ tướng (xiangqi)",
		"caro":     "cờ caro / gomoku",
		"connect4": "connect four / cờ thả cột",
		"reversi":  "reversi / othello",
		"checkers": "cờ đam / checkers",
		"go9":      "cờ vây 9×9 (go)",
	}[gid]
	if label == "" {
		label = gid
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Người chơi vừa đánh một nước %s trên bàn web Vector. ", label))
	b.WriteString(fmt.Sprintf("Hãy gọi %s rồi bình luận ngắn 1 hoặc 2 câu bằng tiếng Việt về nước đi và thế ván. Đừng dùng tool của game khác. Đừng hỏi lại.\n", tool))
	if facts != "" {
		b.WriteString("Nước vừa rồi: ")
		b.WriteString(facts)
		b.WriteString("\n")
	}
	if summary != "" {
		s := summary
		if rs := []rune(s); len(rs) > 400 {
			s = string(rs[:400]) + "…"
		}
		b.WriteString("Tóm tắt: ")
		b.WriteString(s)
	}
	return strings.TrimSpace(b.String())
}
