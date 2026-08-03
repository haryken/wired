package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Compact mailbox Xiangqi (Chinese chess / cờ tướng) for web :80 + Xiaozhi MCP.
// Human = Red; bot = Black after each legal human move.
// Board is 9 files (a-i) x 10 ranks (0-9). sq = rank*9+file.
// Red starts at ranks 0-2 (bottom), Black at ranks 7-9 (top).

type xqPiece byte

const (
	xqEmpty xqPiece = 0
)

func isRedXQ(p xqPiece) bool   { return p >= 'A' && p <= 'Z' }
func isBlackXQ(p xqPiece) bool { return p >= 'a' && p <= 'z' }

func isEnemyXQ(cap, side xqPiece) bool {
	if cap == xqEmpty {
		return false
	}
	return isRedXQ(cap) != isRedXQ(side)
}

func isFriendXQ(cap, side xqPiece) bool {
	if cap == xqEmpty {
		return false
	}
	return isRedXQ(cap) == isRedXQ(side)
}

func xqToLower(p xqPiece) xqPiece {
	if p >= 'A' && p <= 'Z' {
		return p + 32
	}
	return p
}

type xqMove struct {
	From, To int
	Capture  xqPiece
}

type xiangqiGame struct {
	mu          sync.Mutex
	board       [90]xqPiece
	red         bool // true = Red to move
	history     []string
	lastUCI     string
	status      string // playing | check | checkmate | stalemate
	winner      string // red | black | ""
	message     string
	difficulty  string // easy | medium | hard
	botThinking bool
	thinkGen    int
}

var (
	xqMu   sync.Mutex
	xqInst *xiangqiGame
	xqRNG  = rand.New(rand.NewSource(time.Now().UnixNano() + 1))
)

func getXiangqi() *xiangqiGame {
	xqMu.Lock()
	defer xqMu.Unlock()
	if xqInst == nil {
		xqInst = newXiangqiGame()
	}
	return xqInst
}

func newXiangqiGame() *xiangqiGame {
	g := &xiangqiGame{
		red:        true,
		status:     "playing",
		message:    "Ván mới — bạn đi Đỏ. (New game — you are Red.)",
		difficulty: diffMedium,
	}
	g.setFENBoard("rheakaehr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RHEAKAEHR")
	return g
}

// setFENBoard fills the board from a FEN-like board part: 10 rows separated
// by '/', read top (rank 9, Black) to bottom (rank 0, Red), digits 1-9 mean
// that many empty files.
func (g *xiangqiGame) setFENBoard(fenBoard string) {
	for i := range g.board {
		g.board[i] = xqEmpty
	}
	rank, file := 9, 0
	for _, ch := range fenBoard {
		switch {
		case ch == '/':
			rank--
			file = 0
		case ch >= '1' && ch <= '9':
			file += int(ch - '0')
		default:
			g.board[rank*9+file] = xqPiece(ch)
			file++
		}
	}
}

func sqNameXQ(i int) string {
	r, f := i/9, i%9
	return string(rune('a'+f)) + string(rune('0'+r))
}

func parseSqXQ(s string) (int, bool) {
	if len(s) != 2 {
		return 0, false
	}
	f := int(s[0] - 'a')
	r := int(s[1] - '0')
	if f < 0 || f > 8 || r < 0 || r > 9 {
		return 0, false
	}
	return r*9 + f, true
}

func (g *xiangqiGame) boardRowsXQ() []string {
	rows := make([]string, 10)
	for rank := 9; rank >= 0; rank-- {
		var s strings.Builder
		for file := 0; file < 9; file++ {
			p := g.board[rank*9+file]
			if p == xqEmpty {
				s.WriteByte('.')
			} else {
				s.WriteByte(byte(p))
			}
		}
		rows[9-rank] = s.String()
	}
	return rows
}

func (g *xiangqiGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "black"
	if g.red {
		turn = "red"
	}
	hist := append([]string{}, g.history...)
	return map[string]interface{}{
		"board":       g.boardRowsXQ(),
		"turn":        turn,
		"status":      g.status,
		"winner":      g.winner,
		"lastMove":    g.lastUCI,
		"history":     hist,
		"message":     g.message,
		"youAre":      "red",
		"botIs":       "black",
		"difficulty":  normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking,
	}
}

func (g *xiangqiGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "đen (black / bot)"
	if g.red {
		turn = "đỏ (red / người chơi)"
	}
	var b strings.Builder
	b.WriteString("Cờ tướng trên web Vector. Người chơi cầm Đỏ; bot cầm Đen. ")
	b.WriteString("Lượt hiện tại: " + turn + ". Trạng thái: " + g.status + ". ")
	if g.winner != "" {
		b.WriteString("Người thắng: " + g.winner + ". ")
	}
	if g.lastUCI != "" {
		b.WriteString("Nước gần nhất: " + g.lastUCI + ". ")
	}
	b.WriteString("Bàn (hàng 9→0, a→i): ")
	for i, row := range g.boardRowsXQ() {
		b.WriteString(fmt.Sprintf("r%d=%s ", 9-i, row))
	}
	if len(g.history) > 0 {
		b.WriteString("Lịch sử nước đi: " + strings.Join(g.history, " ") + ".")
	}
	return b.String()
}

func (g *xiangqiGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newXiangqiGame()
	ng.difficulty = prev
	xqMu.Lock()
	xqInst = ng
	xqMu.Unlock()
	return ng.snapshot()
}

func (g *xiangqiGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *xiangqiGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func onBoardXQ(r, f int) bool { return r >= 0 && r < 10 && f >= 0 && f < 9 }

// inPalace reports whether (r,f) is inside the palace (files d-f, home three
// ranks) belonging to the given side. Generals and advisors are confined here.
func inPalace(red bool, r, f int) bool {
	if f < 3 || f > 5 {
		return false
	}
	if red {
		return r >= 0 && r <= 2
	}
	return r >= 7 && r <= 9
}

func (g *xiangqiGame) kingSqXQ(red bool) int {
	target := xqPiece('k')
	if red {
		target = 'K'
	}
	for i, p := range g.board {
		if p == target {
			return i
		}
	}
	return -1
}

func (g *xiangqiGame) addStep(moves *[]xqMove, from, to int, side xqPiece) {
	cap := g.board[to]
	if isFriendXQ(cap, side) {
		return
	}
	*moves = append(*moves, xqMove{From: from, To: to, Capture: cap})
}

// genPseudoAt returns pseudo-legal moves for the piece at `from`, based only
// on that piece's own color (ignores whose turn it is on the board). This
// lets attackedXQ() reuse it directly for check / capture detection.
func (g *xiangqiGame) genPseudoAt(from int) []xqMove {
	p := g.board[from]
	if p == xqEmpty {
		return nil
	}
	red := isRedXQ(p)
	var moves []xqMove
	r, f := from/9, from%9
	kind := xqToLower(p)

	switch kind {
	case 'k':
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nr, nf := r+d[0], f+d[1]
			if !inPalace(red, nr, nf) {
				continue
			}
			g.addStep(&moves, from, nr*9+nf, p)
		}
	case 'a':
		for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nr, nf := r+d[0], f+d[1]
			if !inPalace(red, nr, nf) {
				continue
			}
			g.addStep(&moves, from, nr*9+nf, p)
		}
	case 'e':
		for _, d := range [][2]int{{2, 2}, {2, -2}, {-2, 2}, {-2, -2}} {
			nr, nf := r+d[0], f+d[1]
			if !onBoardXQ(nr, nf) {
				continue
			}
			// Elephant eye: the diagonal midpoint must be empty, and the
			// elephant may never cross the river to the opponent's side.
			eyeR, eyeF := r+d[0]/2, f+d[1]/2
			if g.board[eyeR*9+eyeF] != xqEmpty {
				continue
			}
			if red && nr > 4 {
				continue
			}
			if !red && nr < 5 {
				continue
			}
			g.addStep(&moves, from, nr*9+nf, p)
		}
	case 'h':
		for _, d := range [][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
			nr, nf := r+d[0], f+d[1]
			if !onBoardXQ(nr, nf) {
				continue
			}
			// Leg block: the square one step along the long axis of the
			// move must be empty ("horse leg" / 蹩马腿).
			var legR, legF int
			if abs(d[0]) == 2 {
				legR, legF = r+d[0]/2, f
			} else {
				legR, legF = r, f+d[1]/2
			}
			if g.board[legR*9+legF] != xqEmpty {
				continue
			}
			g.addStep(&moves, from, nr*9+nf, p)
		}
	case 'r':
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nr, nf := r+d[0], f+d[1]
			for onBoardXQ(nr, nf) {
				to := nr*9 + nf
				cap := g.board[to]
				if isFriendXQ(cap, p) {
					break
				}
				moves = append(moves, xqMove{From: from, To: to, Capture: cap})
				if cap != xqEmpty {
					break
				}
				nr += d[0]
				nf += d[1]
			}
		}
	case 'c':
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nr, nf := r+d[0], f+d[1]
			screened := false
			for onBoardXQ(nr, nf) {
				to := nr*9 + nf
				cap := g.board[to]
				if !screened {
					if cap == xqEmpty {
						moves = append(moves, xqMove{From: from, To: to, Capture: xqEmpty})
					} else {
						screened = true
					}
				} else if cap != xqEmpty {
					if isEnemyXQ(cap, p) {
						moves = append(moves, xqMove{From: from, To: to, Capture: cap})
					}
					break
				}
				nr += d[0]
				nf += d[1]
			}
		}
	case 'p':
		dir := 1
		crossed := r >= 5
		if !red {
			dir = -1
			crossed = r <= 4
		}
		if onBoardXQ(r+dir, f) {
			g.addStep(&moves, from, (r+dir)*9+f, p)
		}
		if crossed {
			for _, df := range []int{-1, 1} {
				nf := f + df
				if onBoardXQ(r, nf) {
					g.addStep(&moves, from, r*9+nf, p)
				}
			}
		}
	}
	return moves
}

// genPseudo is the turn-aware wrapper used by move generation: it only
// yields moves for the piece whose color matches the side to move.
func (g *xiangqiGame) genPseudo(from int) []xqMove {
	p := g.board[from]
	if p == xqEmpty {
		return nil
	}
	if isRedXQ(p) != g.red {
		return nil
	}
	return g.genPseudoAt(from)
}

func (g *xiangqiGame) attackedXQ(sq int, byRed bool) bool {
	for i := 0; i < 90; i++ {
		p := g.board[i]
		if p == xqEmpty || isRedXQ(p) != byRed {
			continue
		}
		for _, m := range g.genPseudoAt(i) {
			if m.To == sq {
				return true
			}
		}
	}
	return false
}

// kingsFacingXQ implements the "flying general" rule: the two generals may
// never face each other on the same open file.
func (g *xiangqiGame) kingsFacingXQ() bool {
	rk := g.kingSqXQ(true)
	bk := g.kingSqXQ(false)
	if rk < 0 || bk < 0 {
		return false
	}
	rf, bf := rk%9, bk%9
	if rf != bf {
		return false
	}
	rr, br := rk/9, bk/9
	lo, hi := rr, br
	if lo > hi {
		lo, hi = hi, lo
	}
	for r := lo + 1; r < hi; r++ {
		if g.board[r*9+rf] != xqEmpty {
			return false
		}
	}
	return true
}

func (g *xiangqiGame) inCheckXQ(red bool) bool {
	ks := g.kingSqXQ(red)
	if ks < 0 {
		return true
	}
	if g.attackedXQ(ks, !red) {
		return true
	}
	return g.kingsFacingXQ()
}

func (g *xiangqiGame) applyXQ(m xqMove) {
	p := g.board[m.From]
	g.board[m.From] = xqEmpty
	g.board[m.To] = p
	g.red = !g.red
}

func (g *xiangqiGame) undoXQ(m xqMove) {
	g.red = !g.red
	p := g.board[m.To]
	g.board[m.To] = m.Capture
	g.board[m.From] = p
}

func (g *xiangqiGame) legalMovesXQ() []xqMove {
	var out []xqMove
	red := g.red
	for from := 0; from < 90; from++ {
		for _, m := range g.genPseudo(from) {
			g.applyXQ(m)
			ok := !g.inCheckXQ(red)
			g.undoXQ(m)
			if ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func (g *xiangqiGame) refreshStatus() {
	legal := g.legalMovesXQ()
	check := g.inCheckXQ(g.red)
	if len(legal) == 0 {
		if check {
			g.status = "checkmate"
			if g.red {
				g.winner = "black"
				g.message = "Chiếu bí — Đen (bot) thắng. (Checkmate — Black wins.)"
			} else {
				g.winner = "red"
				g.message = "Chiếu bí — Đỏ thắng. (Checkmate — Red wins.)"
			}
		} else {
			g.status = "stalemate"
			g.winner = ""
			g.message = "Hết nước đi — hòa. (Stalemate — draw.)"
		}
		return
	}
	if check {
		g.status = "check"
		g.message = "Chiếu! (Check!)"
	} else {
		g.status = "playing"
		if g.red {
			g.message = "Lượt Đỏ (bạn). (Red to move — your turn.)"
		} else {
			g.message = "Lượt Đen (bot). (Black to move — bot's turn.)"
		}
	}
}

func moveUCIXQ(m xqMove) string {
	return sqNameXQ(m.From) + sqNameXQ(m.To)
}

func (g *xiangqiGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.botThinking {
		g.mu.Unlock()
		return nil, fmt.Errorf("bot thinking")
	}
	if g.status == "checkmate" || g.status == "stalemate" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	from, ok1 := parseSqXQ(uci[0:2])
	to, ok2 := parseSqXQ(uci[2:4])
	if !ok1 || !ok2 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad squares")
	}
	if !g.red {
		g.mu.Unlock()
		return nil, fmt.Errorf("not your turn")
	}
	var chosen *xqMove
	for _, m := range g.legalMovesXQ() {
		if m.From == from && m.To == to {
			mm := m
			chosen = &mm
			break
		}
	}
	if chosen == nil {
		g.mu.Unlock()
		return nil, fmt.Errorf("illegal move")
	}
	youPiece := g.board[chosen.From]
	g.applyXQ(*chosen)
	u := moveUCIXQ(*chosen)
	g.lastUCI = u
	g.history = append(g.history, u)
	youMove := u
	g.refreshStatus()
	needBot := g.status != "checkmate" && g.status != "stalemate" && !g.red
	var thinkGen int
	statusAfterYou := g.status
	winnerAfterYou := g.winner
	if needBot {
		g.botThinking = true
		g.thinkGen++
		thinkGen = g.thinkGen
		g.message = "Bot đang suy nghĩ…"
	}

	turn := "black"
	if g.red {
		turn = "red"
	}
	hist := append([]string{}, g.history...)
	resp := map[string]interface{}{
		"board": g.boardRowsXQ(), "turn": turn,
		"status": g.status, "winner": g.winner, "lastMove": g.lastUCI,
		"history": hist, "message": g.message,
		"youAre": "red", "botIs": "black",
		"youMove": youMove, "botMove": "",
		"youPiece": string(youPiece), "botPiece": "",
		"difficulty":  normalizeGameDifficulty(g.difficulty),
		"botThinking": needBot,
	}
	g.mu.Unlock()

	if youMove != "" {
		stSpeak := statusAfterYou
		winSpeak := winnerAfterYou
		if needBot && stSpeak != "checkmate" && stSpeak != "stalemate" {
			if stSpeak != "check" {
				stSpeak = "playing"
			}
			winSpeak = ""
		}
		// Prefixed summary so Xiaozhi route uses self.xiangqi.* (not self.chess.*).
		xqHint := "Cờ tướng. Nước người chơi: " + youMove + "."
		queueGameSpeak("xiangqi", buildXiangqiSpokenComment(youMove, "", string(youPiece), "", stSpeak, winSpeak), xqHint)
	}

	if needBot {
		go g.runBotThinkXQ(thinkGen)
	}
	return resp, nil
}

func (g *xiangqiGame) runBotThinkXQ(gen int) {
	// Hard spends time SEARCHING, not only sleeping (old: 2s sleep + slow search = >3s still dumb).
	diff := g.getDifficulty()
	switch normalizeGameDifficulty(diff) {
	case diffEasy:
		time.Sleep(900 * time.Millisecond)
	case diffHard:
		time.Sleep(200 * time.Millisecond)
	default:
		time.Sleep(500 * time.Millisecond)
	}
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	botMove := ""
	var botPiece xqPiece
	if g.status != "checkmate" && g.status != "stalemate" && !g.red {
		botPiece = g.botMoveLocked()
		botMove = g.lastUCI
	}
	g.botThinking = false
	if g.red && (g.status == "playing" || g.status == "check") {
		if g.status == "check" {
			g.message = "Đến lượt bạn — đang chiếu!"
		} else {
			g.message = "Đến lượt bạn."
		}
	}
	status := g.status
	winner := g.winner
	summary := "Cờ tướng. Trạng thái: " + g.status + "."
	if g.lastUCI != "" {
		summary += " Nước gần nhất: " + g.lastUCI + "."
	}
	g.mu.Unlock()

	if botMove != "" {
		queueGameSpeak("xiangqi", buildXiangqiSpokenComment("", botMove, "", string(botPiece), status, winner), summary)
	}
}

func (g *xiangqiGame) botMoveLocked() xqPiece {
	legal := g.legalMovesXQ()
	if len(legal) == 0 {
		g.refreshStatus()
		return 0
	}
	var best xqMove
	switch normalizeGameDifficulty(g.difficulty) {
	case diffEasy:
		// Weak: shallow + noise — intentionally makes blunders.
		best = g.pickXQSearch(legal, 2, 35)
	case diffHard:
		best = g.pickXQHard(legal)
	default:
		best = g.pickXQSearch(legal, 3, 4)
	}
	piece := g.board[best.From]
	g.applyXQ(best)
	u := moveUCIXQ(best)
	g.lastUCI = u
	g.history = append(g.history, u)
	g.refreshStatus()
	return piece
}

// staticEvalRedPOV: centipawn-ish; positive = Red better.
func (g *xiangqiGame) staticEvalRedPOV() int {
	score := 0
	for i, p := range g.board {
		if p == xqEmpty {
			continue
		}
		v := materialXQ(p) + xqPST(p, i)
		if isRedXQ(p) {
			score += v
		} else {
			score -= v
		}
	}
	return score
}
func xqPST(p xqPiece, sq int) int {
	r, f := sq/9, sq%9
	center := 0
	if f >= 2 && f <= 6 && r >= 3 && r <= 6 {
		center = 8
	}
	switch xqToLower(p) {
	case 'p':
		if isRedXQ(p) {
			bonus := r * 6
			if r >= 5 {
				bonus += 18
			}
			return bonus + center/2
		}
		bonus := (9 - r) * 6
		if r <= 4 {
			bonus += 18
		}
		return bonus + center/2
	case 'c':
		// Cannon likes open files / mid — discourage baseless back-rank dive without SEE.
		if isRedXQ(p) {
			return center + r*2
		}
		return center + (9-r)*2
	case 'h':
		return center + 4
	case 'r':
		return center
	case 'k':
		if f == 4 {
			return 12
		}
		return 2
	default:
		return center / 2
	}
}

func (g *xiangqiGame) evaluateSTMXQ() int {
	s := g.staticEvalRedPOV()
	if g.red {
		return s
	}
	return -s
}

// xqSEE: static exchange estimate in our material units for the side that moves `m`.
// Positive = good capture / safe place; negative = hanging gift (e.g. pháo nhảy ăn mã rồi bị ăn lại).
func (g *xiangqiGame) xqSEE(m xqMove) int {
	mover := g.board[m.From]
	if mover == xqEmpty {
		return 0
	}
	gain := materialXQ(m.Capture)
	g.applyXQ(m)
	// After move, g.red is the opponent. Can they take our piece on m.To?
	if g.attackedXQ(m.To, g.red) {
		gain -= materialXQ(g.board[m.To])
	}
	// Extra: giving check is OK even at equal trades.
	checking := g.inCheckXQ(g.red)
	g.undoXQ(m)
	if checking && gain >= -materialXQ(mover)/2 {
		// Mild allowance to check.
		if gain < 0 {
			gain += 20
		}
	}
	return gain
}

func (g *xiangqiGame) pickXQEasy(legal []xqMove) xqMove {
	if xqRNG.Intn(100) < 55 {
		return legal[xqRNG.Intn(len(legal))]
	}
	best := legal[xqRNG.Intn(len(legal))]
	bestScore := -99999
	for _, m := range legal {
		score := 0
		if m.Capture != xqEmpty {
			score += 10 + materialXQ(m.Capture)/100
		}
		if score > 0 && xqRNG.Intn(100) < 40 {
			score = 0
		}
		if score > bestScore {
			bestScore = score
			best = m
		}
	}
	if bestScore <= 0 {
		return legal[xqRNG.Intn(len(legal))]
	}
	return best
}

func xqMoveOrder(m xqMove) int {
	if m.Capture != xqEmpty {
		return 1000 + materialXQ(m.Capture)
	}
	return 0
}

func (g *xiangqiGame) orderMovesXQ(legal []xqMove) []xqMove {
	order := make([]xqMove, len(legal))
	copy(order, legal)
	// Sort by capture MVV + SEE bias (higher first).
	type scored struct {
		m xqMove
		s int
	}
	ss := make([]scored, len(order))
	for i, m := range order {
		s := xqMoveOrder(m)
		if m.Capture != xqEmpty {
			s += g.xqSEE(m) // free captures bubble up; hung ones drop
		}
		ss[i] = scored{m, s}
	}
	for i := 0; i < len(ss); i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j].s > ss[i].s {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
	for i := range order {
		order[i] = ss[i].m
	}
	return order
}

func (g *xiangqiGame) pickXQSearch(legal []xqMove, depth int, noisePct int) xqMove {
	if depth < 1 {
		depth = 1
	}
	order := g.orderMovesXQ(legal)
	best := order[0]
	bestScore := -999999
	alpha, beta := -999999, 999999
	for _, m := range order {
		// Soft skip obviously suicidal captures for medium/hard paths that use this helper.
		if noisePct < 20 && m.Capture != xqEmpty && g.xqSEE(m) < -15 {
			continue
		}
		g.applyXQ(m)
		sc := -g.negamaxXQ(depth-1, -beta, -alpha, true)
		g.undoXQ(m)
		if sc > bestScore {
			bestScore = sc
			best = m
		}
		if sc > alpha {
			alpha = sc
		}
	}
	if bestScore <= -999990 {
		// All skipped — fall back raw.
		return g.orderMovesXQ(legal)[0]
	}
	if noisePct > 0 && xqRNG.Intn(100) < noisePct && len(order) > 1 {
		pool := make([]xqMove, 0, 6)
		for _, m := range order {
			if m.Capture != xqEmpty && g.xqSEE(m) < -30 {
				continue
			}
			g.applyXQ(m)
			sc := -g.negamaxXQ(depth-1, -999999, 999999, true)
			g.undoXQ(m)
			if sc >= bestScore-80 {
				pool = append(pool, m)
			}
		}
		if len(pool) > 0 {
			return pool[xqRNG.Intn(len(pool))]
		}
	}
	return best
}

// pickXQHard: depth 3 + SEE-guided root + hang-aware eval. Depth 4 was slow on ARM
// without smarter pruning — felt "thinking long" while still greed-capturing.
func (g *xiangqiGame) pickXQHard(legal []xqMove) xqMove {
	order := g.orderMovesXQ(legal)
	// Hard never plays clearly losing captures at root (except rare checks handled in SEE).
	filtered := make([]xqMove, 0, len(order))
	for _, m := range order {
		if m.Capture != xqEmpty && g.xqSEE(m) < 0 {
			// Allow only if the search still might mate (keep as last resorts).
			continue
		}
		filtered = append(filtered, m)
	}
	if len(filtered) == 0 {
		filtered = order
	} else {
		order = filtered
	}
	best := order[0]
	const maxDepth = 3
	for depth := 1; depth <= maxDepth; depth++ {
		bestScore := -999999
		alpha, beta := -999999, 999999
		localBest := best
		for _, m := range order {
			// Root SEE margin: discourage hung cannon-style dives even if quiet eval lags.
			see := g.xqSEE(m)
			g.applyXQ(m)
			ext := 0
			if g.inCheckXQ(g.red) {
				ext = 1
			}
			sc := -g.negamaxXQ(depth-1+ext, -beta, -alpha, true)
			if see < 0 {
				sc += see * 2 // amplify "don't gift pieces"
			} else if see > 0 {
				sc += see / 2
			}
			if sc > 40000 {
				sc += depth
			}
			g.undoXQ(m)
			if sc > bestScore {
				bestScore = sc
				localBest = m
			}
			if sc > alpha {
				alpha = sc
			}
		}
		best = localBest
		for i, m := range order {
			if m.From == best.From && m.To == best.To {
				order[0], order[i] = order[i], order[0]
				break
			}
		}
	}
	return best
}

func (g *xiangqiGame) negamaxXQ(depth, alpha, beta int, doQ bool) int {
	if depth <= 0 {
		if doQ {
			return g.quiesceXQ(alpha, beta, 3)
		}
		return g.evaluateSTMXQ()
	}
	legal := g.legalMovesXQ()
	if len(legal) == 0 {
		if g.inCheckXQ(g.red) {
			return -50000 + depth
		}
		return 0
	}
	// Order by MVV + crude SEE
	legal = g.orderMovesXQ(legal)
	best := -999999
	for _, m := range legal {
		// Prune obvious hang-captures in deep nodes (keep checks).
		if depth >= 2 && m.Capture != xqEmpty && g.xqSEE(m) < -40 {
			g.applyXQ(m)
			checking := g.inCheckXQ(g.red)
			g.undoXQ(m)
			if !checking {
				continue
			}
		}
		g.applyXQ(m)
		ext := 0
		if depth >= 2 && g.inCheckXQ(g.red) {
			ext = 1
		}
		sc := -g.negamaxXQ(depth-1+ext, -beta, -alpha, doQ)
		g.undoXQ(m)
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
	if best <= -999990 {
		// Everything pruned — re-evaluate without prune.
		return g.evaluateSTMXQ()
	}
	return best
}

func (g *xiangqiGame) quiesceXQ(alpha, beta, qDepth int) int {
	stand := g.evaluateSTMXQ()
	if stand >= beta {
		return beta
	}
	if stand > alpha {
		alpha = stand
	}
	if qDepth <= 0 {
		return stand
	}
	legal := g.orderMovesXQ(g.legalMovesXQ())
	for _, m := range legal {
		if m.Capture == xqEmpty {
			continue
		}
		// δ-pruning style: skip clearly bad captures.
		if g.xqSEE(m) < 0 {
			continue
		}
		g.applyXQ(m)
		sc := -g.quiesceXQ(-beta, -alpha, qDepth-1)
		g.undoXQ(m)
		if sc >= beta {
			return beta
		}
		if sc > alpha {
			alpha = sc
		}
	}
	return alpha
}

// materialXQ — higher resolution values so horse/cannon/rook trades matter.
func materialXQ(p xqPiece) int {
	switch xqToLower(p) {
	case 'p':
		return 100
	case 'a', 'e':
		return 200
	case 'h':
		return 400
	case 'c':
		return 450
	case 'r':
		return 900
	case 'k':
		return 0
	default:
		return 0
	}
}

func (g *xiangqiGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.botThinking || !g.red {
		return []string{}
	}
	ms := g.legalMovesXQ()
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, moveUCIXQ(m))
	}
	return out
}
