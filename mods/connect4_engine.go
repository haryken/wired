package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Connect Four: 7 columns × 6 rows. Human = X drops first, bot = O.

const (
	c4W = 7
	c4H = 6
)

type c4Game struct {
	mu          sync.Mutex
	board       [c4H][c4W]caroCell // reuse caroCell
	humanTurn   bool
	status      string
	winner      string
	lastMove    string
	message     string
	history     []string
	difficulty  string
	botThinking bool
	thinkGen    int
}

var (
	c4Mu   sync.Mutex
	c4Inst *c4Game
)

func getConnect4() *c4Game {
	c4Mu.Lock()
	defer c4Mu.Unlock()
	if c4Inst == nil {
		c4Inst = newC4Game()
	}
	return c4Inst
}

func newC4Game() *c4Game {
	return &c4Game{
		humanTurn:  true,
		status:     "playing",
		message:    "Đến lượt bạn.",
		difficulty: diffMedium,
	}
}

func (g *c4Game) boardRows() []string {
	rows := make([]string, c4H)
	for r := c4H - 1; r >= 0; r-- {
		var b strings.Builder
		for c := 0; c < c4W; c++ {
			switch g.board[r][c] {
			case caroX:
				b.WriteByte('X')
			case caroO:
				b.WriteByte('O')
			default:
				b.WriteByte('.')
			}
		}
		rows[c4H-1-r] = b.String()
	}
	return rows
}

func (g *c4Game) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotUnlocked()
}

func (g *c4Game) snapshotUnlocked() map[string]interface{} {
	turn := "bot"
	if g.humanTurn {
		turn = "human"
	}
	return map[string]interface{}{
		"board": g.boardRows(), "turn": turn, "status": g.status, "winner": g.winner,
		"lastMove": g.lastMove, "history": append([]string{}, g.history...), "message": g.message,
		"youAre": "X", "botIs": "O", "difficulty": normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking, "game": "connect4", "boardW": c4W, "boardH": c4H,
		"placeMode": true, "dropMode": true,
	}
}

func (g *c4Game) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "bot (O)"
	if g.humanTurn {
		turn = "người chơi (X)"
	}
	var b strings.Builder
	b.WriteString("Connect Four (cờ thả cột) 7×6 trên web Vector. Bạn X, bot O. Thắng 4 liên tiếp. ")
	b.WriteString(fmt.Sprintf("Lượt: %s. Trạng thái: %s. ", turn, g.status))
	if g.winner != "" {
		b.WriteString("Thắng: " + g.winner + ". ")
	}
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString("Bàn (trên→dưới): ")
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", c4H-1-i, row))
	}
	if len(g.history) > 0 {
		b.WriteString("Lịch sử: " + strings.Join(g.history, " ") + ".")
	}
	return b.String()
}

func (g *c4Game) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newC4Game()
	ng.difficulty = prev
	c4Mu.Lock()
	c4Inst = ng
	c4Mu.Unlock()
	return ng.snapshot()
}

func (g *c4Game) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *c4Game) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *c4Game) dropRow(col int) int {
	for r := 0; r < c4H; r++ {
		if g.board[r][col] == caroEmpty {
			return r
		}
	}
	return -1
}

func (g *c4Game) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking {
		return nil
	}
	var out []string
	for c := 0; c < c4W; c++ {
		if g.dropRow(c) >= 0 {
			out = append(out, string(rune('a'+c)))
		}
	}
	return out
}

func c4Win(board *[c4H][c4W]caroCell, r, c int, side caroCell) bool {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		n := 1
		for k := 1; k < 4; k++ {
			nr, nc := r+d[1]*k, c+d[0]*k
			if nr < 0 || nr >= c4H || nc < 0 || nc >= c4W || board[nr][nc] != side {
				break
			}
			n++
		}
		for k := 1; k < 4; k++ {
			nr, nc := r-d[1]*k, c-d[0]*k
			if nr < 0 || nr >= c4H || nc < 0 || nc >= c4W || board[nr][nc] != side {
				break
			}
			n++
		}
		if n >= 4 {
			return true
		}
	}
	return false
}

func (g *c4Game) playUCI(uci string) (map[string]interface{}, error) {
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
	uci = strings.ToLower(strings.TrimSpace(uci))
	if len(uci) < 1 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	col := int(uci[0] - 'a')
	if col < 0 || col >= c4W {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad column")
	}
	row := g.dropRow(col)
	if row < 0 {
		g.mu.Unlock()
		return nil, fmt.Errorf("column full")
	}
	g.board[row][col] = caroX
	u := string(rune('a' + col))
	g.lastMove = u
	g.history = append(g.history, "X:"+u)
	if c4Win(&g.board, row, col, caroX) {
		g.status, g.winner, g.message = "win", "human", "Bạn thắng!"
	} else if g.full() {
		g.status, g.message = "draw", "Hoà."
	} else {
		g.humanTurn = false
		g.message = "Bot đang suy nghĩ…"
	}
	needBot := g.status == "playing"
	thinkGen := 0
	if needBot {
		g.botThinking = true
		g.thinkGen++
		thinkGen = g.thinkGen
	}
	youMove := u
	resp := g.snapshotUnlocked()
	resp["youMove"] = youMove
	g.mu.Unlock()
	queueGameSpeak("connect4", buildPlaceSpokenHumanOnly("connect4", youMove, resp["status"].(string), toStr(resp["winner"])), "Connect Four. Nước người chơi cột "+youMove+".")
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

func toStr(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func (g *c4Game) full() bool {
	for c := 0; c < c4W; c++ {
		if g.dropRow(c) >= 0 {
			return false
		}
	}
	return true
}

func (g *c4Game) runBotThink(gen int) {
	time.Sleep(2 * time.Second)
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
		queueGameSpeak("connect4", buildPlaceSpokenBotOnly("connect4", botMove, status, winner), "Connect Four. Bot: "+botMove+".")
	}
}

func (g *c4Game) botMoveLocked() string {
	bestCol, bestSc := -1, -1<<30
	diff := normalizeGameDifficulty(g.difficulty)
	depth := 3
	if diff == diffEasy {
		depth = 2 // old medium-hard: still weak vs human
	} else if diff == diffHard {
		depth = 6
	}
	for c := 0; c < c4W; c++ {
		r := g.dropRow(c)
		if r < 0 {
			continue
		}
		g.board[r][c] = caroO
		var sc int
		if c4Win(&g.board, r, c, caroO) {
			sc = 1_000_000
		} else {
			sc = -c4Negamax(g, depth-1, false, -1<<20, 1<<20)
			// prefer center
			sc += (3 - absInt(c-3)) * 5
		}
		g.board[r][c] = caroEmpty
		if sc > bestSc {
			bestSc, bestCol = sc, c
		}
	}
	if bestCol < 0 {
		g.status, g.message = "draw", "Hoà."
		return ""
	}
	if diff == diffEasy && rand.Intn(100) < 35 {
		// sometimes random legal
		var cols []int
		for c := 0; c < c4W; c++ {
			if g.dropRow(c) >= 0 {
				cols = append(cols, c)
			}
		}
		if len(cols) > 0 {
			bestCol = cols[rand.Intn(len(cols))]
		}
	}
	r := g.dropRow(bestCol)
	g.board[r][bestCol] = caroO
	u := string(rune('a' + bestCol))
	g.lastMove = u
	g.history = append(g.history, "O:"+u)
	if c4Win(&g.board, r, bestCol, caroO) {
		g.status, g.winner, g.message = "win", "bot", "Bot thắng."
	} else if g.full() {
		g.status, g.message = "draw", "Hoà."
	} else {
		g.humanTurn = true
	}
	return u
}

func c4Negamax(g *c4Game, depth int, human bool, alpha, beta int) int {
	if depth == 0 {
		return c4Eval(&g.board)
	}
	side := caroO
	if human {
		side = caroX
	}
	any := false
	best := -1 << 30
	for c := 0; c < c4W; c++ {
		r := g.dropRow(c)
		if r < 0 {
			continue
		}
		any = true
		g.board[r][c] = side
		var sc int
		if c4Win(&g.board, r, c, side) {
			sc = 100000 + depth
		} else {
			sc = -c4Negamax(g, depth-1, !human, -beta, -alpha)
		}
		g.board[r][c] = caroEmpty
		if sc > best {
			best = sc
		}
		if sc > alpha {
			alpha = sc
		}
		if alpha >= beta {
			break
		}
	}
	if !any {
		return 0
	}
	return best
}

func c4Eval(board *[c4H][c4W]caroCell) int {
	// simple: count connectivity favor O (bot)
	sc := 0
	for r := 0; r < c4H; r++ {
		for c := 0; c < c4W; c++ {
			if board[r][c] == caroO {
				sc += 3 + (3 - absInt(c-3))
			} else if board[r][c] == caroX {
				sc -= 3 + (3 - absInt(c-3))
			}
		}
	}
	return sc
}
