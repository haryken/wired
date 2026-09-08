package mods

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GuessNum: Đoán số 1..100, bot chọn số bí mật, tối đa 10 lượt.

const (
	guessMin        = 1
	guessMax        = 100
	guessMaxAttempt = 10
)

type guessHistEntry struct {
	Guess int    `json:"guess"`
	Hint  string `json:"hint"` // higher | lower | exact | miss
	Close bool   `json:"close"`
}

type guessGame struct {
	miniCommon
	secret    int
	low       int
	high      int
	attempts  int
	startedAt time.Time
	lastGuess int
	lastHint  string // higher | lower | exact | lose
	lastClose bool
	hist      []guessHistEntry
}

var (
	guessMu   sync.Mutex
	guessInst *guessGame
)

func getGuessNum() *guessGame {
	guessMu.Lock()
	defer guessMu.Unlock()
	if guessInst == nil {
		guessInst = newGuessGame()
	}
	return guessInst
}

func newGuessGame() *guessGame {
	g := &guessGame{
		secret:    guessMin + rand.Intn(guessMax-guessMin+1),
		low:       guessMin,
		high:      guessMax,
		startedAt: time.Now(),
		hist:      nil,
	}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	g.message, _ = speakf(
		"Tôi đã nghĩ ra một số từ %d đến %d. Hãy đoán xem!",
		"I picked a number from %d to %d. Guess it!",
		guessMin, guessMax,
	)
	return g
}

func guessDiffLabel(diff string) (vi, en string) {
	switch normalizeGameDifficulty(diff) {
	case diffEasy:
		return "Dễ", "Easy"
	case diffHard:
		return "Khó", "Hard"
	default:
		return "Trung bình", "Medium"
	}
}

func guessPick(linesVI, linesEN []string) (vi, en string) {
	if len(linesVI) == 0 {
		return "", ""
	}
	i := rand.Intn(len(linesVI))
	vi = linesVI[i]
	if i < len(linesEN) {
		en = linesEN[i]
	} else {
		en = linesVI[i]
	}
	return
}

func (g *guessGame) commentary(n int, higher bool, exact bool, outOfTries bool) string {
	dist := int(math.Abs(float64(n - g.secret)))
	closeHit := dist > 0 && dist <= 5

	if exact {
		vi, en := guessPick([]string{
			"Xuất sắc! Bạn đã tìm ra con số bí mật.",
			"Tôi biết bạn sẽ làm được!",
			"Chiến thắng rồi!",
			"Bạn thật sự rất giỏi.",
			"Làm thêm một ván nữa nhé!",
			"Đúng rồi! Phải công nhận bạn đoán hay.",
			"Tuyệt vời! Số bí mật đã lộ diện.",
		}, []string{
			"Brilliant! You found the secret number.",
			"I knew you could do it!",
			"Victory!",
			"You're really good at this.",
			"Let's play another round!",
			"Correct! That was a sharp guess.",
			"Awesome! The secret is out.",
		})
		flavor := localizeSpeak(en, vi)
		msg, _ := speakf(
			"%s Số bí mật là %d — bạn thắng sau %d lượt.",
			"%s The number was %d — you won in %d guesses.",
			flavor, g.secret, g.attempts,
		)
		return msg
	}

	if outOfTries {
		msg, _ := speakf(
			"Hết lượt rồi! Số bí mật là %d.",
			"Out of guesses! The number was %d.",
			g.secret,
		)
		return msg
	}

	if closeHit {
		if higher {
			vi, en := guessPick([]string{
				"Rất gần! Cao hơn một chút nữa.",
				"Tôi có cảm giác bạn sắp đúng — thử số lớn hơn.",
				"Chỉ lệch một chút! Hãy đoán cao hơn.",
			}, []string{
				"So close! Go a little higher.",
				"I feel you're almost there — try a bigger number.",
				"Just a bit off! Guess higher.",
			})
			msg, _ := viOrEN(vi, en)
			return msg
		}
		vi, en := guessPick([]string{
			"Rất gần! Thấp hơn một chút nữa.",
			"Tôi có cảm giác bạn sắp đúng — thử số nhỏ hơn.",
			"Chỉ lệch một chút! Hãy đoán thấp hơn.",
		}, []string{
			"So close! Go a little lower.",
			"I feel you're almost there — try a smaller number.",
			"Just a bit off! Guess lower.",
		})
		msg, _ := viOrEN(vi, en)
		return msg
	}

	if higher {
		if dist >= 30 {
			vi, en := guessPick([]string{
				"Bạn còn cách khá xa. Thử một số lớn hơn nhé.",
				"Thấp quá rồi — cao hơn nữa!",
				"Còn xa lắm. Hãy đoán cao hơn.",
			}, []string{
				"You're still quite far. Try a bigger number.",
				"Way too low — go higher!",
				"Still far away. Guess higher.",
			})
			msg, _ := viOrEN(vi, en)
			return msg
		}
		vi, en := guessPick([]string{
			"Cao hơn nữa.",
			"Thử một số lớn hơn nhé.",
			"Hơi thấp — tăng lên một chút.",
		}, []string{
			"Higher.",
			"Try a bigger number.",
			"A bit low — go up a little.",
		})
		msg, _ := viOrEN(vi, en)
		return msg
	}

	if dist >= 30 {
		vi, en := guessPick([]string{
			"Hơi quá rồi. Giảm xuống nhiều hơn.",
			"Cao quá — thử số nhỏ hơn nhé.",
			"Bạn còn cách khá xa về phía trên. Hạ xuống.",
		}, []string{
			"A bit too high. Come down more.",
			"Too high — try a smaller number.",
			"Still far on the high side. Go lower.",
		})
		msg, _ := viOrEN(vi, en)
		return msg
	}
	vi, en := guessPick([]string{
		"Thấp hơn nữa.",
		"Giảm xuống một chút.",
		"Hơi cao — thử số nhỏ hơn.",
	}, []string{
		"Lower.",
		"Come down a little.",
		"A bit high — try smaller.",
	})
	msg, _ := viOrEN(vi, en)
	return msg
}

func (g *guessGame) scoreLocked() int {
	if g.status != "win" {
		return 0
	}
	// Fewer attempts = higher score (max 1000).
	s := 1100 - g.attempts*100
	if s < 100 {
		s = 100
	}
	return s
}

func (g *guessGame) elapsedSecLocked() int {
	if g.startedAt.IsZero() {
		return 0
	}
	return int(time.Since(g.startedAt).Seconds())
}

func (g *guessGame) snapshotLocked() map[string]interface{} {
	hist := make([]map[string]interface{}, 0, len(g.hist))
	for _, h := range g.hist {
		hist = append(hist, map[string]interface{}{
			"guess": h.Guess,
			"hint":  h.Hint,
			"close": h.Close,
		})
	}
	rem := guessMaxAttempt - g.attempts
	if rem < 0 {
		rem = 0
	}
	dVI, dEN := guessDiffLabel(g.difficulty)
	diffLabel := localizeSpeak(dEN, dVI)
	extra := map[string]interface{}{
		"low":             g.low,
		"high":            g.high,
		"min":             guessMin,
		"max":             guessMax,
		"attempts":        g.attempts,
		"maxAttempts":     guessMaxAttempt,
		"remaining":       rem,
		"lastGuess":       g.lastGuess,
		"lastHint":        g.lastHint,
		"lastClose":       g.lastClose,
		"guessHistory":    hist,
		"difficultyLabel": diffLabel,
	}
	if g.status != "playing" {
		extra["secret"] = g.secret
		extra["elapsedSec"] = g.elapsedSecLocked()
		extra["score"] = g.scoreLocked()
	}
	return g.baseSnap("guess", "guess", extra)
}

func (g *guessGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *guessGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Đoán số 1-100. Khoảng còn lại: %d-%d. Đã đoán %d/%d lượt. Trạng thái: %s.",
		g.low, g.high, g.attempts, guessMaxAttempt, g.status)
}

func (g *guessGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	g.mu.Unlock()
	ng := newGuessGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	guessMu.Lock()
	guessInst = ng
	guessMu.Unlock()
	return ng.snapshot()
}

func (g *guessGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *guessGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *guessGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" {
		return nil
	}
	return []string{fmt.Sprintf("<%d-%d>", g.low, g.high)}
}

func (g *guessGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	s := strings.ToLower(strings.TrimSpace(uci))
	s = strings.TrimPrefix(s, "n:")
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < guessMin || n > guessMax {
		g.mu.Unlock()
		return nil, fmt.Errorf("cần một số từ %d đến %d", guessMin, guessMax)
	}

	g.attempts++
	g.moves++
	g.lastMove = strconv.Itoa(n)
	g.lastGuess = n
	g.history = append(g.history, strconv.Itoa(n))

	dist := int(math.Abs(float64(n - g.secret)))
	closeHit := dist > 0 && dist <= 5

	switch {
	case n == g.secret:
		g.status = "win"
		g.winner = "human"
		g.lastHint = "exact"
		g.lastClose = false
		g.hist = append(g.hist, guessHistEntry{Guess: n, Hint: "exact", Close: false})
		g.message = g.commentary(n, false, true, false)

	case n < g.secret:
		if n+1 > g.low {
			g.low = n + 1
		}
		g.lastHint = "higher"
		g.lastClose = closeHit
		g.hist = append(g.hist, guessHistEntry{Guess: n, Hint: "higher", Close: closeHit})
		if g.attempts >= guessMaxAttempt {
			g.status = "lose"
			g.winner = "bot"
			g.lastHint = "lose"
			g.message = g.commentary(n, true, false, true)
		} else {
			g.message = g.commentary(n, true, false, false)
		}

	default: // n > g.secret
		if n-1 < g.high {
			g.high = n - 1
		}
		g.lastHint = "lower"
		g.lastClose = closeHit
		g.hist = append(g.hist, guessHistEntry{Guess: n, Hint: "lower", Close: closeHit})
		if g.attempts >= guessMaxAttempt {
			g.status = "lose"
			g.winner = "bot"
			g.lastHint = "lose"
			g.message = g.commentary(n, false, false, true)
		} else {
			g.message = g.commentary(n, false, false, false)
		}
	}

	resp := g.snapshotLocked()
	msg := g.message
	status := g.status
	g.mu.Unlock()

	hint := fmt.Sprintf("GuessNum: đoán %d", n)
	if status == "win" {
		hint = "Đoán số. Chúc mừng! Bạn thắng."
	} else if status == "lose" {
		hint = "Đoán số. Hết lượt."
	}
	queueGameSpeak("guessnum", msg, hint)
	return resp, nil
}

// NewGuessNum builds the "đoán số" (guess the number) minigame mod.
func NewGuessNum() *genericBoardMod {
	return newGenericBoardMod("GuessNum", "guessnum",
		fmt.Sprintf("Đoán số %d-%d — bot nghĩ số, bạn đoán trong %d lượt", guessMin, guessMax, guessMaxAttempt),
		func() string { return getGuessNum().summaryText() },
		func() map[string]interface{} { return getGuessNum().snapshot() },
		func() map[string]interface{} { return getGuessNum().reset() },
		func(s string) string { return getGuessNum().setDifficulty(s) },
		func() string { return getGuessNum().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getGuessNum().playUCI(u) },
		func() []string { return getGuessNum().legalUCIs() },
		func() (string, string) {
			say, _ := speakf(
				"Ván đoán số mới. Tôi đã nghĩ ra một số từ %d đến %d. Bạn có %d lượt. Chúc may mắn!",
				"New guess-the-number round. I picked a number from %d to %d. You have %d guesses. Good luck!",
				guessMin, guessMax, guessMaxAttempt,
			)
			return say, "Đoán số. Ván mới."
		},
	)
}
