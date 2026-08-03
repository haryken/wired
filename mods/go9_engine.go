package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Micro Go 9×9: capture by liberty, simple Chinese-ish area score at dual pass.
// Human black (X), bot white (O). No superko (positional simple-ko only).

const goN = 9

type goGame struct {
	mu          sync.Mutex
	board       [goN][goN]caroCell
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
	// simple ko: forbidden single recapture square for opponent
	koF, koR int
	hasKo    bool
}

var (
	goMu   sync.Mutex
	goInst *goGame
)

func getGo9() *goGame {
	goMu.Lock()
	defer goMu.Unlock()
	if goInst == nil {
		goInst = newGoGame()
	}
	return goInst
}

func newGoGame() *goGame {
	return &goGame{
		humanTurn: true, status: "playing", message: "Đến lượt bạn (đen).",
		difficulty: diffMedium, koF: -1, koR: -1,
	}
}

func (g *goGame) boardRows() []string {
	rows := make([]string, goN)
	for r := goN - 1; r >= 0; r-- {
		var b strings.Builder
		for f := 0; f < goN; f++ {
			switch g.board[r][f] {
			case caroX:
				b.WriteByte('X')
			case caroO:
				b.WriteByte('O')
			default:
				b.WriteByte('.')
			}
		}
		rows[goN-1-r] = b.String()
	}
	return rows
}

func (g *goGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotUnlocked()
}

func (g *goGame) snapshotUnlocked() map[string]interface{} {
	turn := "bot"
	if g.humanTurn {
		turn = "human"
	}
	return map[string]interface{}{
		"board": g.boardRows(), "turn": turn, "status": g.status, "winner": g.winner,
		"lastMove": g.lastMove, "history": append([]string{}, g.history...), "message": g.message,
		"youAre": "X", "botIs": "O", "difficulty": normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking, "game": "go9", "boardSize": goN, "placeMode": true,
	}
}

func (g *goGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "bot (trắng/O)"
	if g.humanTurn {
		turn = "người chơi (đen/X)"
	}
	var b strings.Builder
	b.WriteString("Cờ vây 9×9 (Go) trên web Vector. Bạn đen, bot trắng. Ăn quân theo khí. Kết thúc khi cả hai pass — chấm điểm vùng đơn giản. ")
	b.WriteString(fmt.Sprintf("Lượt: %s. Trạng thái: %s. ", turn, g.status))
	if g.winner != "" {
		b.WriteString("Thắng: " + g.winner + ". ")
	}
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString("Bàn: ")
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", goN-1-i, row))
	}
	return b.String()
}

func (g *goGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newGoGame()
	ng.difficulty = prev
	goMu.Lock()
	goInst = ng
	goMu.Unlock()
	return ng.snapshot()
}

func (g *goGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *goGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func goFloodGroup(board *[goN][goN]caroCell, f, r int, side caroCell, seen *[goN][goN]bool) (stones [][2]int, libs map[[2]int]bool) {
	libs = map[[2]int]bool{}
	if f < 0 || f >= goN || r < 0 || r >= goN || board[r][f] != side || seen[r][f] {
		return nil, libs
	}
	type p struct{ f, r int }
	q := []p{{f, r}}
	seen[r][f] = true
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		stones = append(stones, [2]int{cur.f, cur.r})
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nf, nr := cur.f+d[0], cur.r+d[1]
			if nf < 0 || nf >= goN || nr < 0 || nr >= goN {
				continue
			}
			if board[nr][nf] == caroEmpty {
				libs[[2]int{nf, nr}] = true
			} else if board[nr][nf] == side && !seen[nr][nf] {
				seen[nr][nf] = true
				q = append(q, p{nf, nr})
			}
		}
	}
	return
}

func goTryPlace(board *[goN][goN]caroCell, f, r int, side caroCell) (ok bool, capt int, koF, koR int, hasKo bool) {
	if board[r][f] != caroEmpty {
		return false, 0, -1, -1, false
	}
	// copy
	var b [goN][goN]caroCell
	b = *board
	b[r][f] = side
	opp := caroO
	if side == caroO {
		opp = caroX
	}
	var seen [goN][goN]bool
	var removed [][2]int
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nf, nr := f+d[0], r+d[1]
		if nf < 0 || nf >= goN || nr < 0 || nr >= goN || b[nr][nf] != opp {
			continue
		}
		if seen[nr][nf] {
			continue
		}
		stones, libs := goFloodGroup(&b, nf, nr, opp, &seen)
		if len(libs) == 0 {
			for _, s := range stones {
				removed = append(removed, s)
			}
		}
	}
	for _, s := range removed {
		b[s[1]][s[0]] = caroEmpty
	}
	// suicide?
	var seen2 [goN][goN]bool
	_, libs := goFloodGroup(&b, f, r, side, &seen2)
	if len(libs) == 0 {
		return false, 0, -1, -1, false
	}
	capt = len(removed)
	if capt == 1 {
		// simple ko point
		hasKo = true
		koF, koR = removed[0][0], removed[0][1]
	}
	*board = b
	return true, capt, koF, koR, hasKo
}

func (g *goGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking {
		return nil
	}
	side := caroO
	if g.humanTurn {
		side = caroX
	}
	out := []string{"pass"}
	for r := 0; r < goN; r++ {
		for f := 0; f < goN; f++ {
			if g.hasKo && f == g.koF && r == g.koR {
				continue
			}
			var b = g.board
			ok, _, _, _, _ := goTryPlace(&b, f, r, side)
			if ok {
				out = append(out, caroSq(f, r))
			}
		}
	}
	return out
}

func (g *goGame) scoreArea() (black, white int) {
	// simple: stones + surrounded empty flood labeled
	var own [goN][goN]int // 0 empty contested, 1 B, 2 W
	for r := 0; r < goN; r++ {
		for f := 0; f < goN; f++ {
			if g.board[r][f] == caroX {
				black++
				own[r][f] = 1
			} else if g.board[r][f] == caroO {
				white++
				own[r][f] = 2
			}
		}
	}
	var vis [goN][goN]bool
	for r := 0; r < goN; r++ {
		for f := 0; f < goN; f++ {
			if g.board[r][f] != caroEmpty || vis[r][f] {
				continue
			}
			// flood empty
			type p struct{ f, r int }
			q := []p{{f, r}}
			vis[r][f] = true
			region := []p{{f, r}}
			touchB, touchW := false, false
			for len(q) > 0 {
				cur := q[0]
				q = q[1:]
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nf, nr := cur.f+d[0], cur.r+d[1]
					if nf < 0 || nf >= goN || nr < 0 || nr >= goN {
						continue
					}
					if g.board[nr][nf] == caroX {
						touchB = true
					} else if g.board[nr][nf] == caroO {
						touchW = true
					} else if !vis[nr][nf] {
						vis[nr][nf] = true
						q = append(q, p{nf, nr})
						region = append(region, p{nf, nr})
					}
				}
			}
			if touchB && !touchW {
				black += len(region)
			} else if touchW && !touchB {
				white += len(region)
			}
		}
	}
	return
}

func (g *goGame) endGame() {
	b, w := g.scoreArea()
	// komi 6.5 for white approx as 6
	w += 6
	g.status = "win"
	if b > w {
		g.winner, g.message = "human", fmt.Sprintf("Bạn thắng ~%d–%d (ước lượng).", b, w)
	} else if w > b {
		g.winner, g.message = "bot", fmt.Sprintf("Bot thắng ~%d–%d (ước lượng).", w, b)
	} else {
		g.status, g.winner, g.message = "draw", "", "Hoà (ước lượng)."
	}
}

func (g *goGame) playUCI(uci string) (map[string]interface{}, error) {
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
	youMove := uci
	if uci == "pass" {
		g.lastMove = "pass"
		g.history = append(g.history, "X:pass")
		g.passStreak++
		g.hasKo = false
		if g.passStreak >= 2 {
			g.endGame()
		} else {
			g.humanTurn = false
			g.message = "Bot đang suy nghĩ…"
		}
	} else {
		f, r, ok := parseCaroSq(uci)
		if !ok || f >= goN || r >= goN {
			g.mu.Unlock()
			return nil, fmt.Errorf("bad move")
		}
		if g.hasKo && f == g.koF && r == g.koR {
			g.mu.Unlock()
			return nil, fmt.Errorf("ko")
		}
		okp, capt, kf, kr, hasKo := goTryPlace(&g.board, f, r, caroX)
		if !okp {
			g.mu.Unlock()
			return nil, fmt.Errorf("illegal (suicide/occupied)")
		}
		u := caroSq(f, r)
		youMove = u
		g.lastMove = u
		if capt > 0 {
			g.history = append(g.history, fmt.Sprintf("X:%s(+%d)", u, capt))
		} else {
			g.history = append(g.history, "X:"+u)
		}
		g.passStreak = 0
		g.hasKo, g.koF, g.koR = hasKo, kf, kr
		g.humanTurn = false
		g.message = "Bot đang suy nghĩ…"
	}
	needBot := g.status == "playing"
	tg := 0
	if needBot {
		g.botThinking = true
		g.thinkGen++
		tg = g.thinkGen
	}
	resp := g.snapshotUnlocked()
	resp["youMove"] = youMove
	g.mu.Unlock()
	queueGameSpeak("go9", buildPlaceSpokenHumanOnly("go9", youMove, toStr(resp["status"]), toStr(resp["winner"])), "Cờ vây 9×9. Bạn: "+youMove+".")
	if needBot {
		go g.runBotThink(tg)
	}
	return resp, nil
}

func (g *goGame) runBotThink(gen int) {
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
		queueGameSpeak("go9", buildPlaceSpokenBotOnly("go9", botMove, status, winner), "Cờ vây. Bot: "+botMove+".")
	}
}

func (g *goGame) botMoveLocked() string {
	type cand struct {
		f, r, sc int
		u        string
	}
	var cs []cand
	for r := 0; r < goN; r++ {
		for f := 0; f < goN; f++ {
			if g.hasKo && f == g.koF && r == g.koR {
				continue
			}
			var b = g.board
			ok, capt, _, _, _ := goTryPlace(&b, f, r, caroO)
			if !ok {
				continue
			}
			sc := capt*100 + 10 - absInt(f-goN/2) - absInt(r-goN/2)
			// avoid edge first few
			cs = append(cs, cand{f, r, sc, caroSq(f, r)})
		}
	}
	if len(cs) == 0 || (normalizeGameDifficulty(g.difficulty) != diffHard && rand.Intn(100) < 8) {
		// pass sometimes
		g.lastMove = "pass"
		g.history = append(g.history, "O:pass")
		g.passStreak++
		g.hasKo = false
		if g.passStreak >= 2 {
			g.endGame()
		} else {
			g.humanTurn = true
		}
		return "pass"
	}
	best := cs[0]
	for _, c := range cs {
		if c.sc > best.sc {
			best = c
		}
	}
	if normalizeGameDifficulty(g.difficulty) == diffEasy && rand.Intn(100) < 45 {
		best = cs[rand.Intn(len(cs))]
	}
	_, capt, kf, kr, hasKo := goTryPlace(&g.board, best.f, best.r, caroO)
	g.lastMove = best.u
	if capt > 0 {
		g.history = append(g.history, fmt.Sprintf("O:%s(+%d)", best.u, capt))
	} else {
		g.history = append(g.history, "O:"+best.u)
	}
	g.passStreak = 0
	g.hasKo, g.koF, g.koR = hasKo, kf, kr
	g.humanTurn = true
	return best.u
}
