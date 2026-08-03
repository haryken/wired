package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// English draughts / checkers 8×8. Human white (bottom ranks 1–3), bot black.

type ckPiece int8

const (
	ckEmpty ckPiece = 0
	ckWM    ckPiece = 1
	ckWK    ckPiece = 2
	ckBM    ckPiece = 3
	ckBK    ckPiece = 4
)

func ckIsWhite(p ckPiece) bool { return p == ckWM || p == ckWK }
func ckIsBlack(p ckPiece) bool { return p == ckBM || p == ckBK }

type ckMove struct {
	from, to int
	caps     []int
	promo    bool
}

type ckGame struct {
	mu          sync.Mutex
	board       [64]ckPiece
	whiteTurn   bool
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
	ckMu   sync.Mutex
	ckInst *ckGame
)

func getCheckers() *ckGame {
	ckMu.Lock()
	defer ckMu.Unlock()
	if ckInst == nil {
		ckInst = newCkGame()
	}
	return ckInst
}

func newCkGame() *ckGame {
	g := &ckGame{whiteTurn: true, status: "playing", message: "Đến lượt bạn.", difficulty: diffMedium}
	for i := 0; i < 64; i++ {
		r, f := i/8, i%8
		if (r+f)%2 == 0 {
			continue
		}
		if r <= 2 {
			g.board[i] = ckWM
		} else if r >= 5 {
			g.board[i] = ckBM
		}
	}
	return g
}

func ckName(i int) string {
	return string(rune('a'+i%8)) + fmt.Sprintf("%d", i/8+1)
}

func ckParse(s string) (int, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 2 {
		return 0, false
	}
	f := int(s[0] - 'a')
	r := int(s[1] - '1')
	if f < 0 || f > 7 || r < 0 || r > 7 {
		return 0, false
	}
	return r*8 + f, true
}

func (g *ckGame) boardRows() []string {
	rows := make([]string, 8)
	for r := 7; r >= 0; r-- {
		var b strings.Builder
		for f := 0; f < 8; f++ {
			switch g.board[r*8+f] {
			case ckWM:
				b.WriteByte('w')
			case ckWK:
				b.WriteByte('W')
			case ckBM:
				b.WriteByte('b')
			case ckBK:
				b.WriteByte('B')
			default:
				b.WriteByte('.')
			}
		}
		rows[7-r] = b.String()
	}
	return rows
}

func (g *ckGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotUnlocked()
}

func (g *ckGame) snapshotUnlocked() map[string]interface{} {
	turn := "bot"
	if g.whiteTurn {
		turn = "human"
	}
	return map[string]interface{}{
		"board": g.boardRows(), "turn": turn, "status": g.status, "winner": g.winner,
		"lastMove": g.lastMove, "history": append([]string{}, g.history...), "message": g.message,
		"youAre": "white", "botIs": "black", "difficulty": normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking, "game": "checkers", "placeMode": false, "moveMode": true,
	}
}

func (g *ckGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "bot (đen)"
	if g.whiteTurn {
		turn = "người chơi (trắng)"
	}
	var b strings.Builder
	b.WriteString("Cờ đam (English draughts/checkers) 8×8 trên web Vector. Bạn trắng, bot đen. Ăn bắt buộc. ")
	b.WriteString(fmt.Sprintf("Lượt: %s. Trạng thái: %s. ", turn, g.status))
	if g.winner != "" {
		b.WriteString("Thắng: " + g.winner + ". ")
	}
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString("Bàn: ")
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", 8-i, row))
	}
	return b.String()
}

func (g *ckGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newCkGame()
	ng.difficulty = prev
	ckMu.Lock()
	ckInst = ng
	ckMu.Unlock()
	return ng.snapshot()
}

func (g *ckGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *ckGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func ckWouldPromo(to int, p ckPiece) bool {
	r := to / 8
	return (p == ckWM && r == 7) || (p == ckBM && r == 0)
}

func (g *ckGame) legalMovesSimple() []ckMove {
	white := g.whiteTurn
	var caps []ckMove
	var boardCopy [64]ckPiece
	copy(boardCopy[:], g.board[:])
	for i := 0; i < 64; i++ {
		p := boardCopy[i]
		if p == ckEmpty {
			continue
		}
		if white && !ckIsWhite(p) {
			continue
		}
		if !white && !ckIsBlack(p) {
			continue
		}
		g.collectCaps(boardCopy, i, i, p, nil, &caps)
	}
	if len(caps) > 0 {
		return caps
	}
	var quiets []ckMove
	for i := 0; i < 64; i++ {
		p := g.board[i]
		if p == ckEmpty {
			continue
		}
		if white && !ckIsWhite(p) {
			continue
		}
		if !white && !ckIsBlack(p) {
			continue
		}
		for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			dr, df := d[0], d[1]
			if p == ckWM && dr < 0 {
				continue
			}
			if p == ckBM && dr > 0 {
				continue
			}
			r, f := i/8, i%8
			tr, tf := r+dr, f+df
			if tr < 0 || tr > 7 || tf < 0 || tf > 7 {
				continue
			}
			to := tr*8 + tf
			if g.board[to] != ckEmpty {
				continue
			}
			quiets = append(quiets, ckMove{from: i, to: to, promo: ckWouldPromo(to, p)})
		}
	}
	return quiets
}

func (g *ckGame) collectCaps(board [64]ckPiece, root, cur int, piece ckPiece, caps []int, out *[]ckMove) {
	r0, f0 := cur/8, cur%8
	for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		dr, df := d[0], d[1]
		if piece == ckWM && dr < 0 {
			continue
		}
		if piece == ckBM && dr > 0 {
			continue
		}
		mr, mf := r0+dr, f0+df
		tr, tf := r0+2*dr, f0+2*df
		if mr < 0 || mr > 7 || mf < 0 || mf > 7 || tr < 0 || tr > 7 || tf < 0 || tf > 7 {
			continue
		}
		mid, to := mr*8+mf, tr*8+tf
		victim := board[mid]
		if board[to] != ckEmpty {
			continue
		}
		if ckIsWhite(piece) && !ckIsBlack(victim) {
			continue
		}
		if ckIsBlack(piece) && !ckIsWhite(victim) {
			continue
		}
		dup := false
		for _, c := range caps {
			if c == mid {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		nb := board
		nb[cur] = ckEmpty
		nb[mid] = ckEmpty
		nb[to] = piece
		newCaps := append(append([]int{}, caps...), mid)
		before := len(*out)
		g.collectCaps(nb, root, to, piece, newCaps, out)
		if len(*out) == before {
			*out = append(*out, ckMove{from: root, to: to, caps: newCaps, promo: ckWouldPromo(to, piece)})
		}
	}
}

func (g *ckGame) applyMove(m ckMove) {
	p := g.board[m.from]
	g.board[m.from] = ckEmpty
	for _, c := range m.caps {
		g.board[c] = ckEmpty
	}
	if m.promo || ckWouldPromo(m.to, p) {
		if ckIsWhite(p) {
			p = ckWK
		} else {
			p = ckBK
		}
	}
	g.board[m.to] = p
}

func (g *ckGame) countSide(white bool) int {
	n := 0
	for i := 0; i < 64; i++ {
		p := g.board[i]
		if white && ckIsWhite(p) {
			n++
		}
		if !white && ckIsBlack(p) {
			n++
		}
	}
	return n
}

func (g *ckGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking {
		return nil
	}
	ms := g.legalMovesSimple()
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, ckName(m.from)+ckName(m.to))
	}
	return out
}

func (g *ckGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.botThinking {
		g.mu.Unlock()
		return nil, fmt.Errorf("bot thinking")
	}
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	if !g.whiteTurn {
		g.mu.Unlock()
		return nil, fmt.Errorf("not your turn")
	}
	uci = strings.ToLower(strings.TrimSpace(uci))
	if len(uci) < 4 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	from, ok1 := ckParse(uci[0:2])
	to, ok2 := ckParse(uci[2:4])
	if !ok1 || !ok2 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad squares")
	}
	var chosen *ckMove
	for _, m := range g.legalMovesSimple() {
		if m.from == from && m.to == to {
			mm := m
			chosen = &mm
			break
		}
	}
	if chosen == nil {
		g.mu.Unlock()
		return nil, fmt.Errorf("illegal move")
	}
	g.applyMove(*chosen)
	u := ckName(chosen.from) + ckName(chosen.to)
	g.lastMove = u
	g.history = append(g.history, u)
	g.whiteTurn = false
	needBot := true
	if g.countSide(false) == 0 || len(g.legalMovesSimple()) == 0 {
		g.status, g.winner, g.message = "win", "human", "Bạn thắng!"
		needBot = false
		g.whiteTurn = true
	} else {
		g.message = "Bot đang suy nghĩ…"
	}
	tg := 0
	if needBot {
		g.botThinking = true
		g.thinkGen++
		tg = g.thinkGen
	}
	resp := g.snapshotUnlocked()
	resp["youMove"] = u
	g.mu.Unlock()
	queueGameSpeak("checkers", buildPlaceSpokenHumanOnly("checkers", u, toStr(resp["status"]), toStr(resp["winner"])), "Cờ đam. Bạn: "+u+".")
	if needBot {
		go g.runBotThink(tg)
	}
	return resp, nil
}

func (g *ckGame) runBotThink(gen int) {
	time.Sleep(2 * time.Second)
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	botMove := ""
	if g.status == "playing" && !g.whiteTurn {
		botMove = g.botMoveLocked()
	}
	g.botThinking = false
	if g.status == "playing" && g.whiteTurn {
		g.message = "Đến lượt bạn."
	}
	status, winner := g.status, g.winner
	g.mu.Unlock()
	if botMove != "" {
		queueGameSpeak("checkers", buildPlaceSpokenBotOnly("checkers", botMove, status, winner), "Cờ đam. Bot: "+botMove+".")
	}
}

func (g *ckGame) botMoveLocked() string {
	ms := g.legalMovesSimple()
	if len(ms) == 0 {
		g.status, g.winner, g.message = "win", "human", "Bạn thắng!"
		return ""
	}
	best := ms[0]
	for _, m := range ms {
		if len(m.caps) > len(best.caps) {
			best = m
		} else if len(m.caps) == len(best.caps) && m.promo && !best.promo {
			best = m
		}
	}
	if normalizeGameDifficulty(g.difficulty) == diffEasy && rand.Intn(100) < 40 {
		best = ms[rand.Intn(len(ms))]
	}
	g.applyMove(best)
	u := ckName(best.from) + ckName(best.to)
	g.lastMove = u
	g.history = append(g.history, u)
	g.whiteTurn = true
	if g.countSide(true) == 0 || len(g.legalMovesSimple()) == 0 {
		g.status, g.winner, g.message = "win", "bot", "Bot thắng."
	}
	return u
}
