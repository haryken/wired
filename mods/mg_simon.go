package mods

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// Simon Says: 4 pads (0-3). The client animates `sequence` whenever
// lastMove == "playback"; the player then repeats it one pad at a time.

type simonGame struct {
	miniCommon
	sequence    []int
	playerInput []int
}

var (
	simonMu   sync.Mutex
	simonInst *simonGame
)

func getSimon() *simonGame {
	simonMu.Lock()
	defer simonMu.Unlock()
	if simonInst == nil {
		simonInst = newSimonGame()
	}
	return simonInst
}

func newSimonGame() *simonGame {
	g := &simonGame{
		sequence: []int{rand.Intn(4)},
	}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	g.lastMove = "playback"
	g.message, _ = viOrEN("Ván Simon mới. Xem dãy rồi lặp lại (0-3).", "New Simon game. Watch the sequence, then repeat it (0-3).")
	return g
}

func (g *simonGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"sequence":  append([]int{}, g.sequence...),
		"playerLen": len(g.playerInput),
		"level":     len(g.sequence),
	}
	return g.baseSnap("simon", "simon", extra)
}

func (g *simonGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *simonGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Simon Says. Mức hiện tại: %d bước. Người chơi đã nhập %d/%d. Trạng thái: %s.",
		len(g.sequence), len(g.playerInput), len(g.sequence), g.status)
}

func (g *simonGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	g.mu.Unlock()
	ng := newSimonGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	simonMu.Lock()
	simonInst = ng
	simonMu.Unlock()
	return ng.snapshot()
}

func (g *simonGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *simonGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *simonGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" {
		return nil
	}
	return []string{"0", "1", "2", "3"}
}

func (g *simonGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	s := strings.ToLower(strings.TrimSpace(uci))
	s = strings.TrimPrefix(s, "pad:")
	pad, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || pad < 0 || pad > 3 {
		g.mu.Unlock()
		return nil, fmt.Errorf("cần một nút từ 0 đến 3")
	}

	g.playerInput = append(g.playerInput, pad)
	g.moves++
	g.lastMove = strconv.Itoa(pad)
	g.history = append(g.history, strconv.Itoa(pad))
	idx := len(g.playerInput) - 1

	if g.playerInput[idx] != g.sequence[idx] {
		g.status = "lose"
		g.winner = "bot"
		level := len(g.sequence)
		seq := append([]int{}, g.sequence...)
		g.playerInput = nil
		g.message, _ = viOrEN(fmt.Sprintf("Sai rồi! Bạn thua ở mức %d.", level), fmt.Sprintf("Wrong! You lost at level %d.", level))
		resp := g.snapshotLocked()
		resp["sequence"] = seq
		msg := g.message
		g.mu.Unlock()
		queueGameSpeak("simon", msg, "Simon: thua ở mức "+strconv.Itoa(level))
		return resp, nil
	}

	var msg string
	if len(g.playerInput) == len(g.sequence) {
		g.sequence = append(g.sequence, rand.Intn(4))
		g.playerInput = nil
		g.lastMove = "playback"
		msg, _ = viOrEN(fmt.Sprintf("Chính xác! Dãy mới có %d bước.", len(g.sequence)), fmt.Sprintf("Correct! New sequence has %d steps.", len(g.sequence)))
	} else {
		msg, _ = viOrEN("Tiếp tục...", "Keep going...")
	}
	g.message = msg

	resp := g.snapshotLocked()
	g.mu.Unlock()

	queueGameSpeak("simon", msg, "Simon")
	return resp, nil
}

// NewSimon builds the Simon Says minigame mod.
func NewSimon() *genericBoardMod {
	return newGenericBoardMod("Simon", "simon",
		"Simon Says — nhớ và lặp lại dãy 4 nút màu",
		func() string { return getSimon().summaryText() },
		func() map[string]interface{} { return getSimon().snapshot() },
		func() map[string]interface{} { return getSimon().reset() },
		func(s string) string { return getSimon().setDifficulty(s) },
		func() string { return getSimon().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getSimon().playUCI(u) },
		func() []string { return getSimon().legalUCIs() },
		func() (string, string) {
			say, _ := viOrEN("Ván Simon mới. Xem dãy rồi lặp lại bằng các nút 0 đến 3.", "New Simon game. Watch the sequence, then repeat it using pads 0 to 3.")
			return say, "Simon. Ván mới."
		},
	)
}
