package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Reversi / Othello 8×8. Human black (X) first, bot white (O).

const revN = 8

type revGame struct {
	mu          sync.Mutex
	board       [revN][revN]caroCell
	humanTurn   bool
	status      string
	winner      string
	lastMove    string
	message     string
	history     []string
	difficulty  string
	botThinking bool
	thinkGen    int
	passStreak  int
}

var (
	revMu   sync.Mutex
	revInst *revGame
)

func getReversi() *revGame {
	revMu.Lock()
	defer revMu.Unlock()
	if revInst == nil {
		revInst = newRevGame()
	}
	return revInst
}

func newRevGame() *revGame {
	g := &revGame{humanTurn: true, status: "playing", message: "Đến lượt bạn.", difficulty: diffMedium}
	// starting position
	g.board[3][3], g.board[4][4] = caroO, caroO
	g.board[3][4], g.board[4][3] = caroX, caroX
	return g
}

func (g *revGame) boardRows() []string {
	rows := make([]string, revN)
	for r := revN - 1; r >= 0; r-- {
		var b strings.Builder
		for f := 0; f < revN; f++ {
			switch g.board[r][f] {
			case caroX:
				b.WriteByte('X')
			case caroO:
				b.WriteByte('O')
			default:
				b.WriteByte('.')
			}
		}
		rows[revN-1-r] = b.String()
	}
	return rows
}

func (g *revGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotUnlocked()
}

func (g *revGame) snapshotUnlocked() map[string]interface{} {
	turn := "bot"
	if g.humanTurn {
		turn = "human"
	}
	hx, ho := g.count()
	return map[string]interface{}{
		"board": g.boardRows(), "turn": turn, "status": g.status, "winner": g.winner,
		"lastMove": g.lastMove, "history": append([]string{}, g.history...), "message": g.message,
		"youAre": "X", "botIs": "O", "difficulty": normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking, "game": "reversi", "placeMode": true,
		"scoreYou": hx, "scoreBot": ho, "boardSize": revN,
	}
}

func (g *revGame) count() (x, o int) {
	for r := 0; r < revN; r++ {
		for f := 0; f < revN; f++ {
			if g.board[r][f] == caroX {
				x++
			} else if g.board[r][f] == caroO {
				o++
			}
		}
	}
	return
}

func (g *revGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	hx, ho := g.count()
	turn := "bot (trắng/O)"
	if g.humanTurn {
		turn = "người chơi (đen/X)"
	}
	var b strings.Builder
	b.WriteString("Reversi/Othello 8×8 trên web Vector. Bạn đen (X), bot trắng (O). ")
	b.WriteString(fmt.Sprintf("Lượt: %s. Điểm X=%d O=%d. Trạng thái: %s. ", turn, hx, ho, g.status))
	if g.winner != "" {
		b.WriteString("Thắng: " + g.winner + ". ")
	}
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString("Bàn: ")
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", revN-1-i, row))
	}
	return b.String()
}

func (g *revGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newRevGame()
	ng.difficulty = prev
	revMu.Lock()
	revInst = ng
	revMu.Unlock()
	return ng.snapshot()
}

func (g *revGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *revGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func revWouldFlip(board *[revN][revN]caroCell, f, r int, side caroCell) [][2]int {
	if board[r][f] != caroEmpty {
		return nil
	}
	opp := caroO
	if side == caroO {
		opp = caroX
	}
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	var flips [][2]int
	for _, d := range dirs {
		var line [][2]int
		cf, cr := f+d[0], r+d[1]
		for cf >= 0 && cf < revN && cr >= 0 && cr < revN && board[cr][cf] == opp {
			line = append(line, [2]int{cf, cr})
			cf += d[0]
			cr += d[1]
		}
		if len(line) > 0 && cf >= 0 && cf < revN && cr >= 0 && cr < revN && board[cr][cf] == side {
			flips = append(flips, line...)
		}
	}
	return flips
}

func (g *revGame) legalFor(side caroCell) []string {
	var out []string
	for r := 0; r < revN; r++ {
		for f := 0; f < revN; f++ {
			if len(revWouldFlip(&g.board, f, r, side)) > 0 {
				out = append(out, caroSq(f, r)) // ranks 0-7
			}
		}
	}
	return out
}

func (g *revGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking {
		return nil
	}
	side := caroO
	if g.humanTurn {
		side = caroX
	}
	return g.legalFor(side)
}

func (g *revGame) apply(f, r int, side caroCell) int {
	flips := revWouldFlip(&g.board, f, r, side)
	g.board[r][f] = side
	for _, p := range flips {
		g.board[p[1]][p[0]] = side
	}
	return len(flips)
}

func (g *revGame) finishIfNeeded() {
	hx, ho := g.count()
	if hx+ho == revN*revN || (len(g.legalFor(caroX)) == 0 && len(g.legalFor(caroO)) == 0) {
		g.status = "win"
		if hx > ho {
			g.winner, g.message = "human", fmt.Sprintf("Bạn thắng %d–%d!", hx, ho)
		} else if ho > hx {
			g.winner, g.message = "bot", fmt.Sprintf("Bot thắng %d–%d.", ho, hx)
		} else {
			g.status, g.winner, g.message = "draw", "", "Hoà."
		}
	}
}

func (g *revGame) playUCI(uci string) (map[string]interface{}, error) {
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
	// pass support
	if strings.EqualFold(uci, "pass") {
		if len(g.legalFor(caroX)) > 0 {
			g.mu.Unlock()
			return nil, fmt.Errorf("cannot pass")
		}
		g.history = append(g.history, "X:pass")
		g.lastMove = "pass"
		g.humanTurn = false
		g.message = "Bot đang suy nghĩ…"
		g.botThinking = true
		g.thinkGen++
		tg := g.thinkGen
		youMove := "pass"
		resp := g.snapshotUnlocked()
		resp["youMove"] = youMove
		g.mu.Unlock()
		queueGameSpeak("reversi", buildPlaceSpokenHumanOnly("reversi", youMove, "playing", ""), "Reversi. Bạn pass.")
		go g.runBotThink(tg)
		return resp, nil
	}
	f, r, ok := parseCaroSq(uci)
	if !ok || f >= revN || r >= revN {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	if len(revWouldFlip(&g.board, f, r, caroX)) == 0 {
		g.mu.Unlock()
		return nil, fmt.Errorf("illegal move")
	}
	n := g.apply(f, r, caroX)
	u := caroSq(f, r)
	g.lastMove = u
	g.history = append(g.history, fmt.Sprintf("X:%s(+%d)", u, n))
	g.finishIfNeeded()
	needBot := g.status == "playing"
	tg := 0
	if needBot {
		g.humanTurn = false
		g.message = "Bot đang suy nghĩ…"
		g.botThinking = true
		g.thinkGen++
		tg = g.thinkGen
	}
	resp := g.snapshotUnlocked()
	resp["youMove"] = u
	g.mu.Unlock()
	queueGameSpeak("reversi", buildPlaceSpokenHumanOnly("reversi", u, toStr(resp["status"]), toStr(resp["winner"])), "Reversi. Bạn: "+u+".")
	if needBot {
		go g.runBotThink(tg)
	}
	return resp, nil
}

func (g *revGame) runBotThink(gen int) {
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
		queueGameSpeak("reversi", buildPlaceSpokenBotOnly("reversi", botMove, status, winner), "Reversi. Bot: "+botMove+".")
	}
}

func (g *revGame) botMoveLocked() string {
	moves := g.legalFor(caroO)
	if len(moves) == 0 {
		// pass
		g.history = append(g.history, "O:pass")
		g.lastMove = "pass"
		if len(g.legalFor(caroX)) == 0 {
			g.finishIfNeeded()
		} else {
			g.humanTurn = true
		}
		return "pass"
	}
	// score moves
	type cand struct {
		f, r, sc int
		u        string
	}
	var cs []cand
	for _, u := range moves {
		f, r, _ := parseCaroSq(u)
		flips := revWouldFlip(&g.board, f, r, caroO)
		sc := len(flips) * 10
		// corners
		if (f == 0 || f == 7) && (r == 0 || r == 7) {
			sc += 200
		}
		// edges
		if f == 0 || f == 7 || r == 0 || r == 7 {
			sc += 20
		}
		// avoid X squares near corners slightly
		cs = append(cs, cand{f, r, sc, u})
	}
	best := cs[0]
	for _, c := range cs {
		if c.sc > best.sc {
			best = c
		}
	}
	if normalizeGameDifficulty(g.difficulty) == diffEasy && rand.Intn(100) < 40 {
		best = cs[rand.Intn(len(cs))]
	}
	n := g.apply(best.f, best.r, caroO)
	g.lastMove = best.u
	g.history = append(g.history, fmt.Sprintf("O:%s(+%d)", best.u, n))
	g.finishIfNeeded()
	if g.status == "playing" {
		if len(g.legalFor(caroX)) == 0 {
			// human must pass — bot may continue? simplify: give turn, UI can pass
			g.humanTurn = true
			g.message = "Bạn không có nước — bấm ô pass không; hệ thống sẽ cho pass tự khi legal rỗng qua bot? "
			// auto-pass human internally once: stay bot if human has no moves
			if len(g.legalFor(caroX)) == 0 {
				// finish or double pass already handled in finishIfNeeded
			}
		} else {
			g.humanTurn = true
		}
	}
	return best.u
}
