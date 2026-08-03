package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Caro freestyle + cấm 2 đầu (double open-3 / double open-4 banned for X / first player).

const caroN = 15

type caroCell int8

const (
	caroEmpty caroCell = 0
	caroX     caroCell = 1 // human
	caroO     caroCell = 2 // bot
)

type caroGame struct {
	mu          sync.Mutex
	board       [caroN][caroN]caroCell
	humanTurn   bool
	status      string // playing | win | draw
	winner      string // human | bot
	lastMove    string
	message     string
	history     []string
	difficulty  string
	botThinking bool
	thinkGen    int
	moves       int
}

var (
	caroMu   sync.Mutex
	caroInst *caroGame
)

func getCaro() *caroGame {
	caroMu.Lock()
	defer caroMu.Unlock()
	if caroInst == nil {
		caroInst = newCaroGame()
	}
	return caroInst
}

func newCaroGame() *caroGame {
	g := &caroGame{
		humanTurn:  true,
		status:     "playing",
		message:    "Đến lượt bạn (X).",
		difficulty: diffMedium,
		history:    nil,
	}
	return g
}

func caroSq(file, rank int) string {
	return string(rune('a'+file)) + fmt.Sprintf("%d", rank)
}

func parseCaroSq(s string) (file, rank int, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 {
		return 0, 0, false
	}
	file = int(s[0] - 'a')
	rank = 0
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, 0, false
		}
		rank = rank*10 + int(s[i]-'0')
	}
	if file < 0 || file >= caroN || rank < 0 || rank >= caroN {
		return 0, 0, false
	}
	return file, rank, true
}

func (g *caroGame) boardRows() []string {
	rows := make([]string, caroN)
	for r := caroN - 1; r >= 0; r-- {
		var b strings.Builder
		for f := 0; f < caroN; f++ {
			switch g.board[r][f] {
			case caroX:
				b.WriteByte('X')
			case caroO:
				b.WriteByte('O')
			default:
				b.WriteByte('.')
			}
		}
		rows[caroN-1-r] = b.String()
	}
	return rows
}

func (g *caroGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "bot"
	if g.humanTurn {
		turn = "human"
	}
	hist := append([]string{}, g.history...)
	return map[string]interface{}{
		"board":       g.boardRows(),
		"turn":        turn,
		"status":      g.status,
		"winner":      g.winner,
		"lastMove":    g.lastMove,
		"history":     hist,
		"message":     g.message,
		"youAre":      "X",
		"botIs":       "O",
		"difficulty":  normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking,
		"game":        "caro",
		"boardSize":   caroN,
		"placeMode":   true,
		"rules":       "freestyle+cam_2_dau",
	}
}

func (g *caroGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "bot (O)"
	if g.humanTurn {
		turn = "người chơi (X)"
	}
	var b strings.Builder
	b.WriteString("Cờ caro (gomoku) freestyle trên web Vector, luật cấm 2 đầu cho quân X (cấm tạo đôi 3 mở hoặc đôi 4 mở). ")
	b.WriteString(fmt.Sprintf("Bàn %dx%d. Lượt: %s. Trạng thái: %s. ", caroN, caroN, turn, g.status))
	if g.winner != "" {
		b.WriteString("Người thắng: " + g.winner + ". ")
	}
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString(fmt.Sprintf("Số nước: %d. Bàn (hàng %d→0): ", g.moves, caroN-1))
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", caroN-1-i, row))
	}
	if len(g.history) > 0 {
		b.WriteString("Lịch sử: " + strings.Join(g.history, " ") + ".")
	}
	return b.String()
}

func (g *caroGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newCaroGame()
	ng.difficulty = prev
	caroMu.Lock()
	caroInst = ng
	caroMu.Unlock()
	return ng.snapshot()
}

func (g *caroGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *caroGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func caroLine(board *[caroN][caroN]caroCell, f, r, df, dr int, side caroCell) int {
	n := 0
	for {
		f += df
		r += dr
		if f < 0 || f >= caroN || r < 0 || r >= caroN || board[r][f] != side {
			break
		}
		n++
	}
	return n
}

func caroWinAt(board *[caroN][caroN]caroCell, f, r int, side caroCell) bool {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		n := 1 + caroLine(board, f, r, d[0], d[1], side) + caroLine(board, f, r, -d[0], -d[1], side)
		if n >= 5 { // freestyle: 5+ wins
			return true
		}
	}
	return false
}

// open ends helpers for one direction through a placed stone.
func caroOpenEnds(board *[caroN][caroN]caroCell, f, r, df, dr int, side caroCell) (length int, openA, openB bool) {
	fwd := caroLine(board, f, r, df, dr, side)
	bwd := caroLine(board, f, r, -df, -dr, side)
	length = 1 + fwd + bwd
	ef, er := f+df*(fwd+1), r+dr*(fwd+1)
	bf, br := f-df*(bwd+1), r-dr*(bwd+1)
	openA = ef >= 0 && ef < caroN && er >= 0 && er < caroN && board[er][ef] == caroEmpty
	openB = bf >= 0 && bf < caroN && br >= 0 && br < caroN && board[br][bf] == caroEmpty
	return
}

// countOpenThrees/Fours scans whole board for continuous runs.
func caroCountOpenPatterns(board *[caroN][caroN]caroCell, side caroCell) (open3, open4 int) {
	seen3 := map[[3]int]bool{}
	seen4 := map[[3]int]bool{}
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for r := 0; r < caroN; r++ {
		for f := 0; f < caroN; f++ {
			if board[r][f] != side {
				continue
			}
			for _, d := range dirs {
				// only start of a run
				pf, pr := f-d[0], r-d[1]
				if pf >= 0 && pf < caroN && pr >= 0 && pr < caroN && board[pr][pf] == side {
					continue
				}
				lenRun := 0
				cf, cr := f, r
				for cf >= 0 && cf < caroN && cr >= 0 && cr < caroN && board[cr][cf] == side {
					lenRun++
					cf += d[0]
					cr += d[1]
				}
				// end cells
				bf, br := f-d[0], r-d[1]
				ef, er := cf, cr
				openB := bf >= 0 && bf < caroN && br >= 0 && br < caroN && board[br][bf] == caroEmpty
				openE := ef >= 0 && ef < caroN && er >= 0 && er < caroN && board[er][ef] == caroEmpty
				midF, midR := f+d[0]*(lenRun/2), r+d[1]*(lenRun/2)
				key := [3]int{midF, midR, d[0]*10 + d[1] + 5}
				if lenRun == 3 && openB && openE {
					if !seen3[key] {
						seen3[key] = true
						open3++
					}
				}
				if lenRun == 4 && (openB || openE) {
					if !seen4[key] {
						seen4[key] = true
						open4++
					}
				}
			}
		}
	}
	return
}

// cấm 2 đầu for X: placing creates ≥2 open threes OR ≥2 open fours.
func caroForbiddenDouble(board *[caroN][caroN]caroCell, side caroCell) bool {
	if side != caroX {
		return false
	}
	o3, o4 := caroCountOpenPatterns(board, side)
	return o3 >= 2 || o4 >= 2
}

func (g *caroGame) boardFull() bool {
	for r := 0; r < caroN; r++ {
		for f := 0; f < caroN; f++ {
			if g.board[r][f] == caroEmpty {
				return false
			}
		}
	}
	return true
}

func (g *caroGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking {
		return nil
	}
	side := caroO
	if g.humanTurn {
		side = caroX
	}
	var out []string
	for r := 0; r < caroN; r++ {
		for f := 0; f < caroN; f++ {
			if g.board[r][f] != caroEmpty {
				continue
			}
			g.board[r][f] = side
			ok := !caroForbiddenDouble(&g.board, side)
			g.board[r][f] = caroEmpty
			if ok {
				out = append(out, caroSq(f, r))
			}
		}
	}
	return out
}

func (g *caroGame) playUCI(uci string) (map[string]interface{}, error) {
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
	f, r, ok := parseCaroSq(uci)
	if !ok {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	if g.board[r][f] != caroEmpty {
		g.mu.Unlock()
		return nil, fmt.Errorf("occupied")
	}
	g.board[r][f] = caroX
	if caroForbiddenDouble(&g.board, caroX) {
		g.board[r][f] = caroEmpty
		g.mu.Unlock()
		return nil, fmt.Errorf("cấm 2 đầu (đôi 3 mở / đôi 4 mở)")
	}
	u := caroSq(f, r)
	g.lastMove = u
	g.history = append(g.history, "X:"+u)
	g.moves++
	youMove := u
	if caroWinAt(&g.board, f, r, caroX) {
		g.status = "win"
		g.winner = "human"
		g.message = "Bạn thắng!"
		g.humanTurn = false
	} else if g.boardFull() {
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
	resp := g.snapshotUnlocked()
	resp["youMove"] = youMove
	resp["botMove"] = ""
	g.mu.Unlock()

	queueGameSpeak("caro", buildPlaceSpokenHumanOnly("caro", youMove, resp["status"].(string), resp["winner"].(string)), "Cờ caro. Nước người chơi: "+youMove+".")
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

func (g *caroGame) snapshotUnlocked() map[string]interface{} {
	turn := "bot"
	if g.humanTurn {
		turn = "human"
	}
	hist := append([]string{}, g.history...)
	win := g.winner
	return map[string]interface{}{
		"board":       g.boardRows(),
		"turn":        turn,
		"status":      g.status,
		"winner":      win,
		"lastMove":    g.lastMove,
		"history":     hist,
		"message":     g.message,
		"youAre":      "X",
		"botIs":       "O",
		"difficulty":  normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking,
		"game":        "caro",
		"boardSize":   caroN,
		"placeMode":   true,
	}
}

func (g *caroGame) runBotThink(gen int) {
	// Short sleep for UI/"thinking" only — search is cheap. Hard almost immediate.
	diff := g.getDifficulty()
	switch diff {
	case diffEasy:
		time.Sleep(700 * time.Millisecond)
	case diffHard:
		time.Sleep(120 * time.Millisecond)
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
		botMove = g.botPlaceLocked()
	}
	g.botThinking = false
	if g.status == "playing" && g.humanTurn {
		g.message = "Đến lượt bạn."
	}
	status, winner := g.status, g.winner
	summary := "Cờ caro. Trạng thái: " + status + ". Nước bot: " + botMove + "."
	g.mu.Unlock()
	if botMove != "" {
		queueGameSpeak("caro", buildPlaceSpokenBotOnly("caro", botMove, status, winner), summary)
	}
}

func (g *caroGame) botPlaceLocked() string {
	// Must hold lock.
	type cand struct {
		f, r, sc int
	}
	hard := normalizeGameDifficulty(g.difficulty) == diffHard
	var cands []cand
	for r := 0; r < caroN; r++ {
		for f := 0; f < caroN; f++ {
			if g.board[r][f] != caroEmpty {
				continue
			}
			// near existing stones only after first moves
			if g.moves > 0 && !caroNear(&g.board, f, r) {
				continue
			}
			sc := caroEvalPlace(&g.board, f, r, caroO)
			if hard {
				// 1-ply: after our move, if opp has an instant win square, devalue
				// (except our own win already scored 1M).
				if sc < 900_000 {
					g.board[r][f] = caroO
					if caroOppHasInstantWin(&g.board, caroX) {
						sc -= 400_000
					}
					g.board[r][f] = caroEmpty
				}
			}
			cands = append(cands, cand{f, r, sc})
		}
	}
	if len(cands) == 0 {
		// any empty
		for r := 0; r < caroN; r++ {
			for f := 0; f < caroN; f++ {
				if g.board[r][f] == caroEmpty {
					cands = append(cands, cand{f, r, 0})
				}
			}
		}
	}
	if len(cands) == 0 {
		g.status = "draw"
		g.message = "Hoà."
		return ""
	}
	// pick by difficulty
	best := cands[0]
	for _, c := range cands {
		if c.sc > best.sc {
			best = c
		}
	}
	switch normalizeGameDifficulty(g.difficulty) {
	case diffEasy:
		// random among top pool — can miss blocks (by design)
		threshold := best.sc - absInt(best.sc)/3 - 50
		if threshold > best.sc-800 {
			// keep some blunders even on high scores, but still prefer not pure worst
			threshold = best.sc - 800
		}
		var pool []cand
		for _, c := range cands {
			if c.sc >= threshold {
				pool = append(pool, c)
			}
		}
		if len(pool) == 0 {
			pool = cands
		}
		best = pool[rand.Intn(len(pool))]
	case diffHard:
		// stick to best (threat-aware)
	default:
		// medium: best score, maybe slight noise only among equal-ish
		var top []cand
		for _, c := range cands {
			if c.sc >= best.sc-5 {
				top = append(top, c)
			}
		}
		if len(top) > 0 {
			best = top[rand.Intn(len(top))]
		}
	}
	g.board[best.r][best.f] = caroO
	u := caroSq(best.f, best.r)
	g.lastMove = u
	g.history = append(g.history, "O:"+u)
	g.moves++
	if caroWinAt(&g.board, best.f, best.r, caroO) {
		g.status = "win"
		g.winner = "bot"
		g.message = "Bot thắng."
		g.humanTurn = false
	} else if g.boardFull() {
		g.status = "draw"
		g.message = "Hoà."
		g.humanTurn = false
	} else {
		g.humanTurn = true
	}
	return u
}

func caroNear(board *[caroN][caroN]caroCell, f, r int) bool {
	for dr := -2; dr <= 2; dr++ {
		for df := -2; df <= 2; df++ {
			nr, nf := r+dr, f+df
			if nr < 0 || nr >= caroN || nf < 0 || nf >= caroN {
				continue
			}
			if board[nr][nf] != caroEmpty {
				return true
			}
		}
	}
	return false
}

// caroOppHasInstantWin: any empty square where side completes 5+.
func caroOppHasInstantWin(board *[caroN][caroN]caroCell, side caroCell) bool {
	for r := 0; r < caroN; r++ {
		for f := 0; f < caroN; f++ {
			if board[r][f] != caroEmpty {
				continue
			}
			board[r][f] = side
			win := caroWinAt(board, f, r, side)
			board[r][f] = caroEmpty
			if win {
				return true
			}
		}
	}
	return false
}

// caroEvalPlace: offense + defense on the SAME square.
//
// Old bug: after placing `side`, it scored opponent shapes on that cell (wrong —
// assumed the placed stone is opponent) and SUBTRACTED the score, so blocking
// open-3s looked worse than extending dead/blocked own lines.
func caroEvalPlace(board *[caroN][caroN]caroCell, f, r int, side caroCell) int {
	opp := caroX
	if side == caroX {
		opp = caroO
	}

	// --- if opponent placed here ---
	board[r][f] = opp
	if caroWinAt(board, f, r, opp) {
		board[r][f] = caroEmpty
		return 500_000 // must block win next
	}
	defThreat := caroThreatScore(board, f, r, opp)
	board[r][f] = caroEmpty

	// --- if we place here ---
	board[r][f] = side
	if side == caroX && caroForbiddenDouble(board, side) {
		board[r][f] = caroEmpty
		return -1_000_000
	}
	if caroWinAt(board, f, r, side) {
		board[r][f] = caroEmpty
		return 1_000_000
	}
	atk := caroThreatScore(board, f, r, side)
	// open patterns after our move (double live-3 for O is strong force)
	o3, o4 := caroCountOpenPatterns(board, side)
	board[r][f] = caroEmpty

	sc := atk + defThreat // defense positive = want to sit on threat squares
	if o4 >= 1 {
		sc += 80_000
	}
	if o3 >= 2 {
		sc += 60_000 // double open-3 force (O allowed; X forbidden elsewhere)
	} else if o3 == 1 {
		sc += 8_000
	}
	// dead lines (0 opens) already score ~0 in caroThreatScore — don't waste moves there
	cx, cy := caroN/2, caroN/2
	sc += 15 - absInt(f-cx) - absInt(r-cy)
	return sc
}

// caroThreatScore assumes board[r][f] already holds `side`.
// Ranks classic gomoku threats; 0-open runs (blocked both ends) stay near 0.
func caroThreatScore(board *[caroN][caroN]caroCell, f, r int, side caroCell) int {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	sc := 0
	for _, d := range dirs {
		length, oa, ob := caroOpenEnds(board, f, r, d[0], d[1], side)
		opens := 0
		if oa {
			opens++
		}
		if ob {
			opens++
		}
		switch {
		case length >= 5:
			sc += 100_000
		case length == 4 && opens == 2:
			// live four — win force
			sc += 90_000
		case length == 4 && opens == 1:
			// half-open four — will win unless blocked
			sc += 45_000
		case length == 4 && opens == 0:
			// dead four (cả 2 đầu bị chặn) — nearly worthless
			sc += 5
		case length == 3 && opens == 2:
			// live three — must answer or becomes live four
			sc += 12_000
		case length == 3 && opens == 1:
			sc += 900
		case length == 3 && opens == 0:
			sc += 3
		case length == 2 && opens == 2:
			sc += 280
		case length == 2 && opens == 1:
			sc += 40
		case length == 2 && opens == 0:
			sc += 1
		case length == 1 && opens == 2:
			sc += 8
		}
	}
	return sc
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
