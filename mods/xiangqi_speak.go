package mods

import "strings"

// Xiangqi auto-comment: reuses the shared chess announce path
// (chessCommentEnabled / getChessCommentMode / queueChessSpeak /
// queueChessSpeakSayTextAlways) defined in chess_speak.go, adding only the
// Vietnamese + English phrasing that is specific to Xiangqi piece names.

func xiangqiPieceNameEN(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "k":
		return "general"
	case "a":
		return "advisor"
	case "e":
		return "elephant"
	case "h":
		return "horse"
	case "r":
		return "chariot"
	case "c":
		return "cannon"
	case "p":
		return "soldier"
	default:
		return "piece"
	}
}

func xiangqiPieceNameVI(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "k":
		return "tướng"
	case "a":
		return "sĩ"
	case "e":
		return "tượng"
	case "h":
		return "mã"
	case "r":
		return "xe"
	case "c":
		return "pháo"
	case "p":
		return "tốt"
	default:
		return "quân cờ"
	}
}

func xiangqiSpeakSq(sq string) string {
	sq = strings.TrimSpace(strings.ToLower(sq))
	if len(sq) < 2 {
		return sq
	}
	return string(sq[0]) + " " + string(sq[1])
}

func xiangqiSpeakMoveEN(piece, uci string) string {
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		return xiangqiPieceNameEN(piece)
	}
	from, to := uci[0:2], uci[2:4]
	return xiangqiPieceNameEN(piece) + " from " + xiangqiSpeakSq(from) + " to " + xiangqiSpeakSq(to)
}

func xiangqiSpeakMoveVI(piece, uci string) string {
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		return xiangqiPieceNameVI(piece)
	}
	from, to := uci[0:2], uci[2:4]
	return xiangqiPieceNameVI(piece) + " từ " + xiangqiSpeakSq(from) + " đến " + xiangqiSpeakSq(to)
}

// xiangqiVietnamese reports whether the active comment mode should speak
// Vietnamese (google_vi / Xiaozhi conversation) instead of English SayText.
func xiangqiVietnamese() bool {
	return getChessCommentMode() == "google_vi"
}

// buildXiangqiCommentEN returns English for Vector Acapela SayText, e.g.
// "You moved chariot from a 0 to a 1. I moved horse from h 9 to g 7."
func buildXiangqiCommentEN(youMove, botMove, youPiece, botPiece, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, "You moved "+xiangqiSpeakMoveEN(youPiece, youMove)+".")
	}
	if botMove != "" {
		parts = append(parts, "I moved "+xiangqiSpeakMoveEN(botPiece, botMove)+".")
	}
	switch status {
	case "checkmate":
		switch winner {
		case "red":
			parts = append(parts, "Checkmate. You win!")
		case "black":
			parts = append(parts, "Checkmate. I win!")
		default:
			parts = append(parts, "Checkmate!")
		}
	case "stalemate":
		parts = append(parts, "Stalemate. Draw.")
	case "check":
		parts = append(parts, "Check!")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// buildXiangqiCommentVI returns Vietnamese for google_vi / Xiaozhi
// conversation mode, e.g. "Bạn đi xe từ a 0 đến a 1. Tôi đi mã từ h 9 đến g 7."
func buildXiangqiCommentVI(youMove, botMove, youPiece, botPiece, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, "Bạn đi "+xiangqiSpeakMoveVI(youPiece, youMove)+".")
	}
	if botMove != "" {
		parts = append(parts, "Tôi đi "+xiangqiSpeakMoveVI(botPiece, botMove)+".")
	}
	switch status {
	case "checkmate":
		switch winner {
		case "red":
			parts = append(parts, "Chiếu bí. Bạn thắng!")
		case "black":
			parts = append(parts, "Chiếu bí. Tôi thắng!")
		default:
			parts = append(parts, "Chiếu bí!")
		}
	case "stalemate":
		parts = append(parts, "Hết nước đi. Hòa.")
	case "check":
		parts = append(parts, "Chiếu!")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// buildXiangqiSpokenComment picks English (saytext) or Vietnamese
// (google_vi / Xiaozhi conversation) phrasing based on the active chess
// comment mode, then delegates to queueChessSpeak in xiangqi.go.
func buildXiangqiSpokenComment(youMove, botMove, youPiece, botPiece, status, winner string) string {
	if xiangqiVietnamese() {
		return buildXiangqiCommentVI(youMove, botMove, youPiece, botPiece, status, winner)
	}
	return buildXiangqiCommentEN(youMove, botMove, youPiece, botPiece, status, winner)
}

func xiangqiNewGameSpeak() string {
	if xiangqiVietnamese() {
		return "Ván mới. Bạn cầm Đỏ. Đến lượt bạn."
	}
	return "New game. You are red. Your move."
}
