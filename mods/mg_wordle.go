package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// Wordle tiếng Việt (không dấu): 5-letter words, 6 guesses.

const (
	wordleWordLen    = 5
	wordleMaxGuesses = 6
)

var wordleWords = []string{
	"NGOAI", "KHONG", "THUOC", "TRUOC", "DUONG", "THANG", "NGANH", "NGUOI",
	"CHUAN", "PHUOC", "TRANG", "CHANH", "THICH", "KHACH", "NHANH", "THANH",
	"HOANG", "QUANG", "THUAN", "NHUNG", "TRONG", "PHONG", "CHONG", "THONG",
	"VUONG", "CUONG", "GIANG", "XUONG", "TUONG", "LUONG", "MUONG", "NUONG",
	"RUONG", "SUONG", "THIEN", "QUYEN", "TUYEN", "NGHIA", "NHIEU", "CHIEU",
	"THIEU", "TRIEU",
}

type wordleGame struct {
	miniCommon
	secret   string
	guesses  []string
	feedback []string
	keyboard map[string]string // letter -> "G"|"Y"|"B" (best seen)
}

var (
	wordleMu   sync.Mutex
	wordleInst *wordleGame
)

func getWordle() *wordleGame {
	wordleMu.Lock()
	defer wordleMu.Unlock()
	if wordleInst == nil {
		wordleInst = newWordleGame()
	}
	return wordleInst
}

func newWordleGame() *wordleGame {
	g := &wordleGame{
		secret:   wordleWords[rand.Intn(len(wordleWords))],
		keyboard: map[string]string{},
	}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	g.message, _ = speakf(
		"Ván Wordle mới. Đoán từ %d chữ cái tiếng Việt không dấu. Bạn có %d lượt.",
		"New Wordle. Guess the %d-letter Vietnamese word (no diacritics). You have %d guesses.",
		wordleWordLen, wordleMaxGuesses,
	)
	return g
}

// wordleRank maps a feedback letter to its priority for keyboard hints.
func wordleRank(s string) int {
	switch s {
	case "G":
		return 3
	case "Y":
		return 2
	case "B":
		return 1
	default:
		return 0
	}
}

// wordleComputeFeedback runs the classic two-pass Wordle scoring so
// duplicate letters are handled correctly (green pass first, then yellow).
func wordleComputeFeedback(secret, guess string) string {
	n := len(secret)
	fb := make([]byte, n)
	remaining := map[byte]int{}
	for i := 0; i < n; i++ {
		if guess[i] == secret[i] {
			fb[i] = 'G'
		} else {
			remaining[secret[i]]++
		}
	}
	for i := 0; i < n; i++ {
		if fb[i] == 'G' {
			continue
		}
		c := guess[i]
		if remaining[c] > 0 {
			fb[i] = 'Y'
			remaining[c]--
		} else {
			fb[i] = 'B'
		}
	}
	return string(fb)
}

func (g *wordleGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"guesses":    append([]string{}, g.guesses...),
		"feedback":   append([]string{}, g.feedback...),
		"maxGuesses": wordleMaxGuesses,
		"wordLen":    wordleWordLen,
		"keyboard":   g.keyboard,
	}
	if g.status != "playing" {
		extra["secret"] = g.secret
	}
	return g.baseSnap("wordle", "wordle", extra)
}

func (g *wordleGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *wordleGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Wordle tiếng Việt không dấu, từ %d chữ cái, tối đa %d lượt. ", wordleWordLen, wordleMaxGuesses))
	b.WriteString(fmt.Sprintf("Trạng thái: %s. Đã đoán %d lượt. ", g.status, len(g.guesses)))
	for i, gu := range g.guesses {
		b.WriteString(fmt.Sprintf("%s(%s) ", gu, g.feedback[i]))
	}
	if g.status != "playing" {
		b.WriteString("Từ đúng: " + g.secret + ".")
	}
	return b.String()
}

func (g *wordleGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	g.mu.Unlock()
	ng := newWordleGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	wordleMu.Lock()
	wordleInst = ng
	wordleMu.Unlock()
	return ng.snapshot()
}

func (g *wordleGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *wordleGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *wordleGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" {
		return nil
	}
	return []string{"<5 chữ cái A-Z>"}
}

func (g *wordleGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	guess := strings.ToUpper(strings.TrimSpace(uci))
	guess = strings.TrimPrefix(guess, "GUESS:")
	guess = strings.TrimSpace(guess)
	if len(guess) != wordleWordLen {
		g.mu.Unlock()
		return nil, fmt.Errorf("cần đoán đúng %d chữ cái", wordleWordLen)
	}
	for i := 0; i < len(guess); i++ {
		if guess[i] < 'A' || guess[i] > 'Z' {
			g.mu.Unlock()
			return nil, fmt.Errorf("chỉ dùng chữ cái A-Z, không dấu")
		}
	}

	fb := wordleComputeFeedback(g.secret, guess)
	g.guesses = append(g.guesses, guess)
	g.feedback = append(g.feedback, fb)
	g.moves++
	g.lastMove = guess
	g.history = append(g.history, guess+":"+fb)
	for i := 0; i < len(guess); i++ {
		ch := string(guess[i])
		st := string(fb[i])
		if wordleRank(st) > wordleRank(g.keyboard[ch]) {
			g.keyboard[ch] = st
		}
	}

	won := guess == g.secret
	if won {
		g.status = "win"
		g.winner = "human"
		g.message, _ = viOrEN("Chính xác! Bạn thắng.", "Correct! You win.")
	} else if len(g.guesses) >= wordleMaxGuesses {
		g.status = "lose"
		g.winner = "bot"
		g.message, _ = speakf("Hết lượt. Từ đúng là %s.", "Out of guesses. The word was %s.", g.secret)
	} else {
		remain := wordleMaxGuesses - len(g.guesses)
		g.message, _ = speakf("Còn %d lượt đoán.", "%d guesses left.", remain)
	}

	resp := g.snapshotLocked()
	msg := g.message
	g.mu.Unlock()

	queueGameSpeak("wordle", msg, "Wordle: "+guess+" -> "+fb)
	return resp, nil
}

// NewWordle builds the Wordle (tiếng Việt) minigame mod.
func NewWordle() *genericBoardMod {
	return newGenericBoardMod("Wordle", "wordle",
		"Wordle tiếng Việt (không dấu) — đoán từ 5 chữ cái trong 6 lượt",
		func() string { return getWordle().summaryText() },
		func() map[string]interface{} { return getWordle().snapshot() },
		func() map[string]interface{} { return getWordle().reset() },
		func(s string) string { return getWordle().setDifficulty(s) },
		func() string { return getWordle().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getWordle().playUCI(u) },
		func() []string { return getWordle().legalUCIs() },
		func() (string, string) {
			say, _ := speakf(
				"Ván Wordle mới. Đoán từ tiếng Việt %d chữ cái, không dấu. Bạn có %d lượt.",
				"New Wordle. Guess the %d-letter Vietnamese word, no diacritics. You have %d guesses.",
				wordleWordLen, wordleMaxGuesses,
			)
			return say, "Wordle. Ván mới."
		},
	)
}
