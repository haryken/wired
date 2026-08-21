package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Tic-tac-toe: 3x3, human X first, bot O. UCI squares a0..c2 (file a-c, rank 0-2).

const tttN = 3

type tttGame struct {
	miniCommon
	board [tttN][tttN]caroCell
}

var (
	tttMu   sync.Mutex
	tttInst *tttGame
)

func getTTT() *tttGame {
	tttMu.Lock()
	defer tttMu.Unlock()
	if tttInst == nil {
		tttInst = newTTTGame()
	}
	return tttInst
}

func newTTTGame() *tttGame {
	g := &tttGame{}
	g.humanTurn = true
	g.status = "playing"
	g.difficulty = diffMedium
	g.message = "Đến lượt bạn (X)."
	return g
}

// tttLines enumerates the 8 winning lines as [rank][file] pairs.
var tttLines = [8][3][2]int{
	{{0, 0}, {0, 1}, {0, 2}},
	{{1, 0}, {1, 1}, {1, 2}},
	{{2, 0}, {2, 1}, {2, 2}},
	{{0, 0}, {1, 0}, {2, 0}},
	{{0, 1}, {1, 1}, {2, 1}},
	{{0, 2}, {1, 2}, {2, 2}},
	{{0, 0}, {1, 1}, {2, 2}},
	{{0, 2}, {1, 1}, {2, 0}},
}

func tttWinner(b *[tttN][tttN]caroCell) caroCell {
	for _, ln := range tttLines {
		a := b[ln[0][0]][ln[0][1]]
		if a != caroEmpty && a == b[ln[1][0]][ln[1][1]] && a == b[ln[2][0]][ln[2][1]] {
			return a
		}
	}
	return caroEmpty
}

func tttFull(b *[tttN][tttN]caroCell) bool {
	for r := 0; r < tttN; r++ {
		for f := 0; f < tttN; f++ {
			if b[r][f] == caroEmpty {
				return false
			}
		}
	}
	return true
}

// tttMinimaxScore is a perfect-play evaluation, positive favors bot (O).
func tttMinimaxScore(b *[tttN][tttN]caroCell, depth int, botToMove bool) int {
	if w := tttWinner(b); w == caroO {
		return 10 - depth
	} else if w == caroX {
		return depth - 10
	}
	if tttFull(b) {
		return 0
	}
	if botToMove {
		best := -1000
		for r := 0; r < tttN; r++ {
			for f := 0; f < tttN; f++ {
				if b[r][f] != caroEmpty {
					continue
				}
				b[r][f] = caroO
				sc := tttMinimaxScore(b, depth+1, false)
				b[r][f] = caroEmpty
				if sc > best {
					best = sc
				}
			}
		}
		return best
	}
	best := 1000
	for r := 0; r < tttN; r++ {
		for f := 0; f < tttN; f++ {
			if b[r][f] != caroEmpty {
				continue
			}
			b[r][f] = caroX
			sc := tttMinimaxScore(b, depth+1, true)
			b[r][f] = caroEmpty
			if sc < best {
				best = sc
			}
		}
	}
	return best
}

// boardRows renders top rank2 → rank0, matching the caro/connect4 convention.
func (g *tttGame) boardRows() []string {
	rows := make([]string, tttN)
	for r := tttN - 1; r >= 0; r-- {
		var b strings.Builder
		for f := 0; f < tttN; f++ {
			switch g.board[r][f] {
			case caroX:
				b.WriteByte('X')
			case caroO:
				b.WriteByte('O')
			default:
				b.WriteByte('.')
			}
		}
		rows[tttN-1-r] = b.String()
	}
	return rows
}

func (g *tttGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *tttGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"board": g.boardRows(), "boardW": tttN, "boardH": tttN,
		"youAre": "X", "botIs": "O",
	}
	return g.baseSnap("tictactoe", "tictactoe", extra)
}

func (g *tttGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "bot (O)"
	if g.humanTurn {
		turn = "người chơi (X)"
	}
	var b strings.Builder
	b.WriteString("Tic-tac-toe 3x3 trên web Vector. Bạn X đi trước, bot O. ")
	b.WriteString(fmt.Sprintf("Lượt: %s. Trạng thái: %s. ", turn, g.status))
	if g.winner != "" {
		b.WriteString("Thắng: " + g.winner + ". ")
	}
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString(fmt.Sprintf("Bàn (hàng %d→0): ", tttN-1))
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", tttN-1-i, row))
	}
	if len(g.history) > 0 {
		b.WriteString("Lịch sử: " + strings.Join(g.history, " ") + ".")
	}
	return b.String()
}

func (g *tttGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newTTTGame()
	ng.difficulty = prev
	tttMu.Lock()
	tttInst = ng
	tttMu.Unlock()
	return ng.snapshot()
}

func (g *tttGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *tttGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *tttGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking || !g.humanTurn {
		return nil
	}
	var out []string
	for r := 0; r < tttN; r++ {
		for f := 0; f < tttN; f++ {
			if g.board[r][f] == caroEmpty {
				out = append(out, miniSqName(f, r))
			}
		}
	}
	return out
}

func (g *tttGame) playUCI(uci string) (map[string]interface{}, error) {
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
	f, r, ok := miniParseSq(uci)
	if !ok || f < 0 || f >= tttN || r < 0 || r >= tttN {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	if g.board[r][f] != caroEmpty {
		g.mu.Unlock()
		return nil, fmt.Errorf("occupied")
	}
	g.board[r][f] = caroX
	u := miniSqName(f, r)
	g.lastMove = u
	g.history = append(g.history, "X:"+u)
	g.moves++
	if tttWinner(&g.board) == caroX {
		g.status = "win"
		g.winner = "human"
		g.message = "Bạn thắng!"
		g.humanTurn = false
	} else if tttFull(&g.board) {
		g.status = "draw"
		g.message = "Hoà."
		g.humanTurn = false
	} else {
		g.humanTurn = false
		g.message = "Bot đang suy nghĩ…"
	}
	needBot := g.status == "playing"
	var thinkGen int
	if needBot {
		g.botThinking = true
		g.thinkGen++
		thinkGen = g.thinkGen
	}
	resp := g.snapshotLocked()
	resp["youMove"] = u
	resp["botMove"] = ""
	g.mu.Unlock()

	statusStr, _ := resp["status"].(string)
	winnerStr, _ := resp["winner"].(string)
	queueGameSpeak("tictactoe", buildPlaceSpokenHumanOnly("tictactoe", u, statusStr, winnerStr), "Tic tac toe. Nước người chơi: "+u+".")
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

func (g *tttGame) runBotThink(gen int) {
	diff := g.getDifficulty()
	switch diff {
	case diffEasy:
		time.Sleep(500 * time.Millisecond)
	case diffHard:
		time.Sleep(200 * time.Millisecond)
	default:
		time.Sleep(350 * time.Millisecond)
	}
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	botMove := ""
	if g.status == "playing" && !g.humanTurn {
		botMove = g.botMoveLocked()
	}
	g.botThinking = false
	if g.status == "playing" && g.humanTurn {
		g.message = "Đến lượt bạn."
	}
	status, winner := g.status, g.winner
	g.mu.Unlock()
	if botMove != "" {
		queueGameSpeak("tictactoe", buildPlaceSpokenBotOnly("tictactoe", botMove, status, winner), "Tic tac toe. Bot: "+botMove+".")
	}
}

// botMoveLocked picks bot's (O) move given current difficulty. Must hold lock.
func (g *tttGame) botMoveLocked() string {
	type cand struct{ f, r, sc int }
	var cands []cand
	for r := 0; r < tttN; r++ {
		for f := 0; f < tttN; f++ {
			if g.board[r][f] != caroEmpty {
				continue
			}
			g.board[r][f] = caroO
			sc := tttMinimaxScore(&g.board, 1, false)
			g.board[r][f] = caroEmpty
			cands = append(cands, cand{f, r, sc})
		}
	}
	if len(cands) == 0 {
		g.status = "draw"
		g.message = "Hoà."
		return ""
	}
	diff := normalizeGameDifficulty(g.difficulty)
	best := cands[0]
	for _, c := range cands {
		if c.sc > best.sc {
			best = c
		}
	}
	var pick cand
	switch diff {
	case diffHard:
		pick = best
	case diffEasy:
		// Prefer an immediate win or block if trivially available, else mostly random.
		win := false
		for _, c := range cands {
			if c.sc >= 9 {
				pick = c
				win = true
				break
			}
		}
		if !win && rand.Intn(100) < 70 {
			pick = cands[rand.Intn(len(cands))]
		} else if !win {
			pick = best
		}
	default: // medium: perfect play with occasional noise among near-best
		var top []cand
		for _, c := range cands {
			if c.sc >= best.sc-2 {
				top = append(top, c)
			}
		}
		if rand.Intn(100) < 20 && len(cands) > 1 {
			pick = cands[rand.Intn(len(cands))]
		} else {
			pick = top[rand.Intn(len(top))]
		}
	}
	g.board[pick.r][pick.f] = caroO
	u := miniSqName(pick.f, pick.r)
	g.lastMove = u
	g.history = append(g.history, "O:"+u)
	g.moves++
	if tttWinner(&g.board) == caroO {
		g.status = "win"
		g.winner = "bot"
		g.message = "Bot thắng."
		g.humanTurn = false
	} else if tttFull(&g.board) {
		g.status = "draw"
		g.message = "Hoà."
		g.humanTurn = false
	} else {
		g.humanTurn = true
	}
	return u
}

// NewTicTacToe wires the tic-tac-toe mod using the generic board-game HTTP surface.
func NewTicTacToe() *genericBoardMod {
	return newGenericBoardMod("TicTacToe", "tictactoe",
		"Tic-tac-toe 3x3 web game — Vector comments; Xiaozhi MCP",
		func() string { return getTTT().summaryText() },
		func() map[string]interface{} { return getTTT().snapshot() },
		func() map[string]interface{} { return getTTT().reset() },
		func(s string) string { return getTTT().setDifficulty(s) },
		func() string { return getTTT().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getTTT().playUCI(u) },
		func() []string { return getTTT().legalUCIs() },
		func() (string, string) {
			if chessPreferVIText() {
				return "Ván tic tac toe mới. Bạn cầm X, đi trước. Đến lượt bạn.", "Tic tac toe. Ván mới."
			}
			return "New tic-tac-toe game. You are X and go first. Your move.", "Tic tac toe. Ván mới."
		},
	)
}
