package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Memory (concentration): 4x4, 8 pairs A-H. UCI square a0-d3 flips a card.
// Human flips 2 cards; on mismatch turn passes to the bot, which also flips 2.

const memN = 4

type memPick struct{ file, rank int }

type memoryGame struct {
	miniCommon
	cards      [memN][memN]byte
	matched    [memN][memN]bool
	faceUp     [memN][memN]bool
	pending    *memPick
	known      map[[2]int]byte // remembered symbols from prior flips (both sides)
	pairsHuman int
	pairsBot   int
}

var (
	memMu   sync.Mutex
	memInst *memoryGame
)

func getMemory() *memoryGame {
	memMu.Lock()
	defer memMu.Unlock()
	if memInst == nil {
		memInst = newMemoryGame()
	}
	return memInst
}

func newMemoryGame() *memoryGame {
	g := &memoryGame{known: map[[2]int]byte{}}
	g.humanTurn = true
	g.status = "playing"
	g.difficulty = diffMedium
	g.message = "Lật 2 ô để tìm cặp giống nhau."
	g.shuffleCards()
	return g
}

func (g *memoryGame) shuffleCards() {
	symbols := []byte{'A', 'A', 'B', 'B', 'C', 'C', 'D', 'D', 'E', 'E', 'F', 'F', 'G', 'G', 'H', 'H'}
	rand.Shuffle(len(symbols), func(i, j int) { symbols[i], symbols[j] = symbols[j], symbols[i] })
	idx := 0
	for r := 0; r < memN; r++ {
		for c := 0; c < memN; c++ {
			g.cards[r][c] = symbols[idx]
			idx++
		}
	}
}

func (g *memoryGame) rememberLocked(r, c int) {
	if g.known == nil {
		g.known = map[[2]int]byte{}
	}
	g.known[[2]int{r, c}] = g.cards[r][c]
}

func (g *memoryGame) boardRows() []string {
	rows := make([]string, memN)
	for r := 0; r < memN; r++ {
		var b strings.Builder
		for c := 0; c < memN; c++ {
			switch {
			case g.matched[r][c]:
				b.WriteByte(g.cards[r][c] - 'A' + 'a')
			case g.faceUp[r][c]:
				b.WriteByte(g.cards[r][c])
			default:
				b.WriteByte('?')
			}
		}
		rows[r] = b.String()
	}
	return rows
}

func (g *memoryGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *memoryGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"board": g.boardRows(), "boardW": memN, "boardH": memN,
		"pairsHuman": g.pairsHuman, "pairsBot": g.pairsBot,
	}
	return g.baseSnap("memory", "memory", extra)
}

func (g *memoryGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Trò chơi trí nhớ 4x4 (8 cặp) trên web Vector. Bạn: %d cặp, Bot: %d cặp. Trạng thái: %s.",
		g.pairsHuman, g.pairsBot, g.status)
}

func (g *memoryGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newMemoryGame()
	ng.difficulty = prev
	memMu.Lock()
	memInst = ng
	memMu.Unlock()
	return ng.snapshot()
}

func (g *memoryGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *memoryGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *memoryGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking || !g.humanTurn {
		return nil
	}
	var out []string
	for r := 0; r < memN; r++ {
		for c := 0; c < memN; c++ {
			if !g.matched[r][c] && !g.faceUp[r][c] {
				out = append(out, miniSqName(c, r))
			}
		}
	}
	return out
}

func (g *memoryGame) checkGameOverLocked() {
	if g.pairsHuman+g.pairsBot == memN*memN/2 {
		if g.pairsHuman > g.pairsBot {
			g.status, g.winner, g.message = "win", "human", "Bạn thắng!"
		} else if g.pairsBot > g.pairsHuman {
			g.status, g.winner, g.message = "win", "bot", "Bot thắng."
		} else {
			g.status, g.winner, g.message = "draw", "", "Hoà."
		}
	}
}

func (g *memoryGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.botThinking {
		g.mu.Unlock()
		return nil, fmt.Errorf("bot thinking")
	}
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	if !g.humanTurn {
		g.mu.Unlock()
		return nil, fmt.Errorf("not your turn")
	}
	file, rank, ok := miniParseSq(uci)
	if !ok || file < 0 || file >= memN || rank < 0 || rank >= memN {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	if g.matched[rank][file] || g.faceUp[rank][file] {
		g.mu.Unlock()
		return nil, fmt.Errorf("cell unavailable")
	}
	u := miniSqName(file, rank)
	g.faceUp[rank][file] = true
	g.rememberLocked(rank, file)

	if g.pending == nil {
		g.pending = &memPick{file, rank}
		g.lastMove = u
		g.history = append(g.history, "H1:"+u)
		g.message = "Lật ô thứ hai."
		resp := g.snapshotLocked()
		resp["youMove"] = u
		resp["flipStage"] = 1
		g.mu.Unlock()
		queueGameSpeak("memory", "Bạn lật "+u+".", "Trí nhớ. Lật "+u+".")
		return resp, nil
	}

	p := g.pending
	g.pending = nil
	u1 := miniSqName(p.file, p.rank)
	matched := g.cards[p.rank][p.file] == g.cards[rank][file]
	var speakMsg string
	if matched {
		g.matched[p.rank][p.file] = true
		g.matched[rank][file] = true
		g.pairsHuman++
		g.message = "Trùng khớp! Bạn đi tiếp."
		speakMsg = "Trùng cặp " + u1 + " và " + u + ". Bạn đi tiếp."
	} else {
		g.faceUp[p.rank][p.file] = false
		g.faceUp[rank][file] = false
		g.message = "Không trùng. Đến lượt bot."
		speakMsg = "Không trùng " + u1 + " và " + u + "."
	}
	g.lastMove = u
	g.history = append(g.history, "H2:"+u)
	g.moves++
	g.checkGameOverLocked()
	gameOver := g.status != "playing"

	needBot := false
	var thinkGen int
	if !gameOver && !matched {
		g.humanTurn = false
		g.botThinking = true
		g.thinkGen++
		thinkGen = g.thinkGen
		needBot = true
	}
	resp := g.snapshotLocked()
	resp["youMove"] = u
	resp["matched"] = matched
	resp["flipStage"] = 2
	resp["pair"] = []string{u1, u}
	g.mu.Unlock()

	statusStr, _ := resp["status"].(string)
	winnerStr, _ := resp["winner"].(string)
	queueGameSpeak("memory", buildPlaceSpokenHumanOnly("memory", speakMsg, statusStr, winnerStr), "Trí nhớ. "+speakMsg)
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

// memoryChance is the % chance the bot uses its remembered-card knowledge.
func (g *memoryGame) memoryChance() int {
	switch normalizeGameDifficulty(g.difficulty) {
	case diffEasy:
		return 25
	case diffHard:
		return 100
	default:
		return 60
	}
}

func (g *memoryGame) findKnownPairLocked() (r1, c1, r2, c2 int, ok bool) {
	bySym := map[byte][][2]int{}
	for pos, sym := range g.known {
		if g.matched[pos[0]][pos[1]] || g.faceUp[pos[0]][pos[1]] {
			continue
		}
		bySym[sym] = append(bySym[sym], pos)
	}
	for _, positions := range bySym {
		if len(positions) >= 2 {
			return positions[0][0], positions[0][1], positions[1][0], positions[1][1], true
		}
	}
	return 0, 0, 0, 0, false
}

func (g *memoryGame) randomUnrevealedLocked(excludeR, excludeC int) (int, int) {
	var cands [][2]int
	for r := 0; r < memN; r++ {
		for c := 0; c < memN; c++ {
			if g.matched[r][c] || g.faceUp[r][c] {
				continue
			}
			if r == excludeR && c == excludeC {
				continue
			}
			cands = append(cands, [2]int{r, c})
		}
	}
	if len(cands) == 0 {
		return 0, 0
	}
	p := cands[rand.Intn(len(cands))]
	return p[0], p[1]
}

// botPickLocked chooses the bot's next flip. excludeR<0 means this is the
// first flip of the turn; otherwise it is the second, excluding the first.
func (g *memoryGame) botPickLocked(excludeR, excludeC int) (int, int) {
	useMemory := rand.Intn(100) < g.memoryChance()
	if excludeR < 0 {
		if useMemory {
			if r1, c1, _, _, ok := g.findKnownPairLocked(); ok {
				return r1, c1
			}
		}
		return g.randomUnrevealedLocked(-1, -1)
	}
	if useMemory {
		target := g.cards[excludeR][excludeC]
		for pos, sym := range g.known {
			if sym == target && !(pos[0] == excludeR && pos[1] == excludeC) &&
				!g.matched[pos[0]][pos[1]] && !g.faceUp[pos[0]][pos[1]] {
				return pos[0], pos[1]
			}
		}
	}
	return g.randomUnrevealedLocked(excludeR, excludeC)
}

func (g *memoryGame) runBotThink(gen int) {
	time.Sleep(700 * time.Millisecond)
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	if g.status != "playing" || g.humanTurn {
		g.botThinking = false
		g.mu.Unlock()
		return
	}
	r1, c1 := g.botPickLocked(-1, -1)
	g.faceUp[r1][c1] = true
	g.rememberLocked(r1, c1)
	u1 := miniSqName(c1, r1)
	g.mu.Unlock()

	time.Sleep(500 * time.Millisecond)

	g.mu.Lock()
	if gen != g.thinkGen {
		g.mu.Unlock()
		return
	}
	r2, c2 := g.botPickLocked(r1, c1)
	g.faceUp[r2][c2] = true
	g.rememberLocked(r2, c2)
	u2 := miniSqName(c2, r2)
	matched := g.cards[r1][c1] == g.cards[r2][c2]
	var msg string
	if matched {
		g.matched[r1][c1] = true
		g.matched[r2][c2] = true
		g.pairsBot++
		msg = "Bot lật " + u1 + " và " + u2 + ", trùng cặp!"
		g.message = "Bot trùng cặp, bot đi tiếp."
	} else {
		g.faceUp[r1][c1] = false
		g.faceUp[r2][c2] = false
		msg = "Bot lật " + u1 + " và " + u2 + ", không trùng."
		g.message = "Đến lượt bạn."
	}
	g.moves++
	g.lastMove = u2
	g.history = append(g.history, "B:"+u1+","+u2)
	g.checkGameOverLocked()
	gameOver := g.status != "playing"

	if !gameOver && matched {
		g.thinkGen++
		thinkGen2 := g.thinkGen
		g.mu.Unlock()
		queueGameSpeak("memory", msg, "Trí nhớ. "+msg)
		go g.runBotThink(thinkGen2)
		return
	}
	g.botThinking = false
	if !gameOver {
		g.humanTurn = true
	}
	status, winner := g.status, g.winner
	g.mu.Unlock()
	queueGameSpeak("memory", buildPlaceSpokenBotOnly("memory", msg, status, winner), "Trí nhớ. "+msg)
}

// NewMemory wires the memory mod using the generic board-game HTTP surface.
func NewMemory() *genericBoardMod {
	return newGenericBoardMod("Memory", "memory",
		"Memory (lật thẻ đôi) 4x4 web game — Vector comments; Xiaozhi MCP",
		func() string { return getMemory().summaryText() },
		func() map[string]interface{} { return getMemory().snapshot() },
		func() map[string]interface{} { return getMemory().reset() },
		func(s string) string { return getMemory().setDifficulty(s) },
		func() string { return getMemory().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getMemory().playUCI(u) },
		func() []string { return getMemory().legalUCIs() },
		func() (string, string) {
			return speakNew(
				"New memory game. Flip two cards to find matching pairs.",
				"Ván trí nhớ mới. Lật 2 ô để tìm cặp giống nhau.",
			)
		},
	)
}
