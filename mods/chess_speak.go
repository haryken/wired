package mods

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Auto-comment after web chess moves: wired → /run/vic-cloud/chess-announce →
// vic-cloud SayText (mode saytext) or Xiaozhi WSS inject (mode xiaozhi).

const chessAnnouncePath = "/run/vic-cloud/chess-announce"

const (
	chessModeSayText  = "saytext"
	chessModeXiaozhi  = "xiaozhi"
	chessModeGoogleVI = "google_vi"
)

var (
	chessCommentOn   int32 = 1
	chessSpeakMu     sync.Mutex
	chessSpeakBusy   int32
	chessSpeakQueued string

	// 0 = saytext, 1 = xiaozhi, 2 = google_vi
	chessCommentMode int32
)

type chessAnnouncePayload struct {
	Mode    string `json:"mode"`
	Say     string `json:"say"`
	Prompt  string `json:"prompt,omitempty"`
	Summary string `json:"summary,omitempty"`
	Lang    string `json:"lang,omitempty"`
}

func chessCommentEnabled() bool {
	return atomic.LoadInt32(&chessCommentOn) == 1
}

func setChessCommentEnabled(on bool) {
	if on {
		atomic.StoreInt32(&chessCommentOn, 1)
	} else {
		atomic.StoreInt32(&chessCommentOn, 0)
	}
}

func chessXiaozhiAvailable() bool {
	cfg, err := loadXiaozhiCfg()
	if err != nil {
		return false
	}
	return cfg.Enabled
}

func chessGoogleVIAvailable() bool {
	// Always on — language is chosen in the Games lobby dropdown.
	return true
}

// chessGoogleTTSLang is the Google Translate TTS language code (default vi).
func chessGoogleTTSLang() string {
	cfg, err := loadXiaozhiCfg()
	if err != nil {
		return "vi"
	}
	return normalizeGoogleTTSLang(cfg.GameGoogleTTSLang)
}

// chessPreferVIText is true only for Google TTS + Vietnamese.
// Other Google languages use gameSpeakLang() + localized templates.
func chessPreferVIText() bool {
	return gameSpeakLang() == "vi"
}

func normalizeGoogleTTSLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "", "vi", "vi-vn":
		return "vi"
	case "zh", "zh-cn", "cn", "chinese":
		return "zh-CN"
	case "en", "en-us", "en-gb":
		return "en"
	case "it", "it-it":
		return "it"
	case "ru", "ru-ru":
		return "ru"
	case "fr", "fr-fr":
		return "fr"
	case "de", "de-de":
		return "de"
	case "es", "es-es":
		return "es"
	case "pt", "pt-br", "pt-pt":
		return "pt"
	default:
		return "vi"
	}
}

func setChessGoogleTTSLang(lang string) string {
	lang = normalizeGoogleTTSLang(lang)
	cfg, _ := loadXiaozhiCfg()
	ensureXiaozhiDefaults(&cfg)
	cfg.GameGoogleTTSVI = true
	cfg.GameGoogleTTSLang = lang
	_ = saveXiaozhiCfg(cfg)
	return lang
}

func getChessCommentMode() string {
	switch atomic.LoadInt32(&chessCommentMode) {
	case 1:
		if chessXiaozhiAvailable() {
			return chessModeXiaozhi
		}
	case 2:
		if chessGoogleVIAvailable() {
			return chessModeGoogleVI
		}
	}
	return chessModeSayText
}

func setChessCommentMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case chessModeXiaozhi, "conversation", "ws":
		if chessXiaozhiAvailable() {
			atomic.StoreInt32(&chessCommentMode, 1)
			return chessModeXiaozhi
		}
		atomic.StoreInt32(&chessCommentMode, 0)
		return chessModeSayText
	case chessModeGoogleVI, "google", "google-vi":
		if chessGoogleVIAvailable() {
			atomic.StoreInt32(&chessCommentMode, 2)
			return chessModeGoogleVI
		}
		atomic.StoreInt32(&chessCommentMode, 0)
		return chessModeSayText
	default:
		atomic.StoreInt32(&chessCommentMode, 0)
		return chessModeSayText
	}
}

func queueChessSpeak(say, summary string) {
	queueGameSpeak("", say, summary)
}

// queueGameSpeak is queueChessSpeak with explicit gameID for Xiaozhi MCP routing.
func queueGameSpeak(gameID, say, summary string) {
	say = strings.TrimSpace(say)
	if say == "" || !chessCommentEnabled() {
		return
	}
	mode := getChessCommentMode()
	// google_vi: comment text matches GameGoogleTTSLang (vi, zh-CN, it, …).
	prompt := ""
	if mode == chessModeXiaozhi {
		prompt = buildXiaozhiGamePrompt(gameID, say, summary)
	}
	lang := ""
	if mode == chessModeGoogleVI {
		lang = chessGoogleTTSLang()
	}
	payload, err := json.Marshal(chessAnnouncePayload{
		Mode:    mode,
		Say:     say,
		Prompt:  prompt,
		Summary: strings.TrimSpace(summary),
		Lang:    lang,
	})
	if err != nil {
		log.Println("[Chess] announce marshal:", err)
		return
	}
	chessSpeakMu.Lock()
	chessSpeakQueued = string(payload)
	chessSpeakMu.Unlock()
	if !atomic.CompareAndSwapInt32(&chessSpeakBusy, 0, 1) {
		return
	}
	go drainChessSpeak()
}

// queueChessSpeakSayTextAlways forces Acapela SayText (e.g. exit game), even if
// auto-comment is off or Xiaozhi conversation mode is selected.
func queueChessSpeakSayTextAlways(say string) {
	say = strings.TrimSpace(say)
	if say == "" {
		return
	}
	payload, err := json.Marshal(chessAnnouncePayload{
		Mode: chessModeSayText,
		Say:  say,
	})
	if err != nil {
		log.Println("[Chess] announce marshal:", err)
		return
	}
	chessSpeakMu.Lock()
	chessSpeakQueued = string(payload)
	chessSpeakMu.Unlock()
	if !atomic.CompareAndSwapInt32(&chessSpeakBusy, 0, 1) {
		return
	}
	go drainChessSpeak()
}

func drainChessSpeak() {
	defer atomic.StoreInt32(&chessSpeakBusy, 0)
	for {
		chessSpeakMu.Lock()
		text := chessSpeakQueued
		chessSpeakQueued = ""
		chessSpeakMu.Unlock()
		if text == "" {
			return
		}
		if err := writeChessAnnounce(text); err != nil {
			log.Println("[Chess] announce write:", err)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func writeChessAnnounce(text string) error {
	_ = os.MkdirAll("/run/vic-cloud", 0775)
	tmp := chessAnnouncePath + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, chessAnnouncePath)
}

func pieceName(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "k":
		return "king"
	case "q":
		return "queen"
	case "r":
		return "rook"
	case "b":
		return "bishop"
	case "n":
		return "knight"
	case "p":
		return "pawn"
	default:
		return "piece"
	}
}

func speakSq(sq string) string {
	sq = strings.TrimSpace(strings.ToLower(sq))
	if len(sq) < 2 {
		return sq
	}
	return string(sq[0]) + " " + string(sq[1])
}

func speakMove(piece, uci string) string {
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		return pieceName(piece)
	}
	from, to := uci[0:2], uci[2:4]
	s := pieceName(piece) + " from " + speakSq(from) + " to " + speakSq(to)
	if len(uci) >= 5 {
		s += ", promote to " + pieceName(string(uci[4]))
	}
	return s
}

// buildChessComment returns English for Vector Acapela, with piece names.
func buildChessComment(youMove, botMove, youPiece, botPiece, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, "You moved "+speakMove(youPiece, youMove)+".")
	}
	if botMove != "" {
		parts = append(parts, "I moved "+speakMove(botPiece, botMove)+".")
	}
	switch status {
	case "checkmate":
		if winner == "white" {
			parts = append(parts, "Checkmate. You win!")
		} else if winner == "black" {
			parts = append(parts, "Checkmate. I win!")
		} else {
			parts = append(parts, "Checkmate!")
		}
	case "stalemate":
		parts = append(parts, "Stalemate. Draw.")
	case "check":
		parts = append(parts, "Check!")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func pieceNameVI(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "k":
		return "vua"
	case "q":
		return "hậu"
	case "r":
		return "xe"
	case "b":
		return "tượng"
	case "n":
		return "mã"
	case "p":
		return "tốt"
	default:
		return "quân"
	}
}

func speakMoveVI(piece, uci string) string {
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		return pieceNameVI(piece)
	}
	from, to := uci[0:2], uci[2:4]
	s := pieceNameVI(piece) + " từ " + speakSq(from) + " đến " + speakSq(to)
	if len(uci) >= 5 {
		s += ", phong " + pieceNameVI(string(uci[4]))
	}
	return s
}

func buildChessCommentVI(youMove, botMove, youPiece, botPiece, status, winner string) string {
	var parts []string
	if youMove != "" {
		parts = append(parts, "Bạn đi "+speakMoveVI(youPiece, youMove)+".")
	}
	if botMove != "" {
		parts = append(parts, "Tôi đi "+speakMoveVI(botPiece, botMove)+".")
	}
	switch status {
	case "checkmate":
		if winner == "white" {
			parts = append(parts, "Chiếu hết. Bạn thắng!")
		} else if winner == "black" {
			parts = append(parts, "Chiếu hết. Tôi thắng!")
		} else {
			parts = append(parts, "Chiếu hết!")
		}
	case "stalemate":
		parts = append(parts, "Hết nước. Hòa.")
	case "check":
		parts = append(parts, "Chiếu!")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// buildChessSpokenComment matches SayText (English) or the Google TTS language.
func buildChessSpokenComment(youMove, botMove, youPiece, botPiece, status, winner string) string {
	lang := gameSpeakLang()
	if lang == "en" {
		return buildChessComment(youMove, botMove, youPiece, botPiece, status, winner)
	}
	if lang == "vi" {
		return buildChessCommentVI(youMove, botMove, youPiece, botPiece, status, winner)
	}
	return buildChessCommentForLang(youMove, botMove, youPiece, botPiece, status, winner)
}

func speakGameName(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "chess", "co-vua", "cờ vua":
		return "chess"
	case "xiangqi", "co-tuong", "cờ tướng":
		return "xiangqi"
	case "caro", "gomoku", "cờ caro":
		return "caro"
	case "connect4", "connect-four", "cờ thả", "connect four":
		return "connect four"
	case "reversi", "othello":
		return "reversi"
	case "checkers", "draughts", "cờ đam":
		return "checkers"
	case "go9", "go", "cờ vây", "weiqi":
		return "go"
	default:
		if id == "" {
			return "game"
		}
		return strings.ToLower(id)
	}
}
