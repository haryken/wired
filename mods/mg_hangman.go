package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// Hangman tiếng Việt (không dấu), maxWrong = 7.

const hangmanMaxWrong = 7

var hangmanWords = []string{
	"NGAY", "BUOI", "CHAO", "TUOI", "SACH", "HOA", "MEO", "CHO", "VIT", "CAY",
	"NUOC", "LUA", "DAT", "TROI", "MAY", "MUA", "NANG", "GIO", "BIEN", "NUI",
	"SONG", "RUNG", "CHIM", "TOM", "CUA", "GAO", "BANH", "KEO", "SUA", "TRA",
}

type hangmanGame struct {
	miniCommon
	secret  string
	guessed map[string]bool
	wrong   []string
}

var (
	hangmanMu   sync.Mutex
	hangmanInst *hangmanGame
)

func getHangman() *hangmanGame {
	hangmanMu.Lock()
	defer hangmanMu.Unlock()
	if hangmanInst == nil {
		hangmanInst = newHangmanGame()
	}
	return hangmanInst
}

func newHangmanGame() *hangmanGame {
	g := &hangmanGame{
		secret:  hangmanWords[rand.Intn(len(hangmanWords))],
		guessed: map[string]bool{},
	}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	g.message, _ = viOrEN("Ván treo cổ mới. Đoán từng chữ cái.", "New hangman game. Guess a letter.")
	return g
}

func (g *hangmanGame) maskedLocked() string {
	var b strings.Builder
	for _, r := range g.secret {
		ch := string(r)
		if g.guessed[ch] {
			b.WriteString(ch)
		} else {
			b.WriteString("_")
		}
	}
	return b.String()
}

func (g *hangmanGame) wonLocked() bool {
	for _, r := range g.secret {
		if !g.guessed[string(r)] {
			return false
		}
	}
	return true
}

func (g *hangmanGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"masked":     g.maskedLocked(),
		"wrong":      append([]string{}, g.wrong...),
		"wrongCount": len(g.wrong),
		"maxWrong":   hangmanMaxWrong,
		"wordLen":    len(g.secret),
	}
	if g.status != "playing" {
		extra["secret"] = g.secret
	}
	return g.baseSnap("hangman", "hangman", extra)
}

func (g *hangmanGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *hangmanGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Hangman tiếng Việt không dấu. Từ: %s. Sai %d/%d. Trạng thái: %s.",
		g.maskedLocked(), len(g.wrong), hangmanMaxWrong, g.status)
}

func (g *hangmanGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	g.mu.Unlock()
	ng := newHangmanGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	hangmanMu.Lock()
	hangmanInst = ng
	hangmanMu.Unlock()
	return ng.snapshot()
}

func (g *hangmanGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *hangmanGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *hangmanGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" {
		return nil
	}
	var out []string
	for c := 'A'; c <= 'Z'; c++ {
		ch := string(c)
		if !g.guessed[ch] {
			out = append(out, strings.ToLower(ch))
		}
	}
	return out
}

func (g *hangmanGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	letter := strings.ToUpper(strings.TrimSpace(uci))
	letter = strings.TrimPrefix(letter, "LETTER:")
	letter = strings.TrimSpace(letter)
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		g.mu.Unlock()
		return nil, fmt.Errorf("cần đúng 1 chữ cái A-Z")
	}
	if g.guessed[letter] {
		g.mu.Unlock()
		return nil, fmt.Errorf("đã đoán chữ này rồi")
	}
	g.guessed[letter] = true
	g.moves++
	g.lastMove = letter

	hit := strings.Contains(g.secret, letter)
	if hit {
		g.history = append(g.history, letter+":hit")
		g.message, _ = viOrEN("Đúng rồi!", "Correct!")
	} else {
		g.wrong = append(g.wrong, letter)
		g.history = append(g.history, letter+":miss")
		g.message, _ = viOrEN("Sai rồi!", "Wrong!")
	}

	if g.wonLocked() {
		g.status = "win"
		g.winner = "human"
		g.message, _ = viOrEN("Bạn đoán đúng cả từ! Bạn thắng.", "You guessed the whole word! You win.")
	} else if len(g.wrong) >= hangmanMaxWrong {
		g.status = "lose"
		g.winner = "bot"
		g.message, _ = viOrEN(fmt.Sprintf("Bạn thua! Từ đúng là %s.", g.secret), fmt.Sprintf("You lose! The word was %s.", g.secret))
	}

	resp := g.snapshotLocked()
	msg := g.message
	g.mu.Unlock()

	queueGameSpeak("hangman", msg, "Hangman: "+letter)
	return resp, nil
}

// NewHangman builds the Hangman (tiếng Việt) minigame mod.
func NewHangman() *genericBoardMod {
	return newGenericBoardMod("Hangman", "hangman",
		"Hangman tiếng Việt (không dấu) — đoán chữ cái, tối đa 7 lần sai",
		func() string { return getHangman().summaryText() },
		func() map[string]interface{} { return getHangman().snapshot() },
		func() map[string]interface{} { return getHangman().reset() },
		func(s string) string { return getHangman().setDifficulty(s) },
		func() string { return getHangman().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getHangman().playUCI(u) },
		func() []string { return getHangman().legalUCIs() },
		func() (string, string) {
			say, _ := viOrEN("Ván treo cổ mới. Đoán từng chữ cái tiếng Việt không dấu.", "New hangman game. Guess a letter of the Vietnamese word (no diacritics).")
			return say, "Hangman. Ván mới."
		},
	)
}
