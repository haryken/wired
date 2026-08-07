package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Battleship: 8x8, both fleets auto-placed randomly. Player fires on the
// enemy via UCI a0-h7; bot fires back after every player shot.

const bsN = 8

var bsShipSizes = []int{5, 4, 3, 3, 2}

type bsShipState struct {
	cells [][2]int
	hits  int
}

type bsPoint struct{ r, c int }

type battleshipGame struct {
	miniCommon
	enemyGrid  [bsN][bsN]bool
	enemyIdx   [bsN][bsN]int
	enemyFleet []*bsShipState
	enemyHit   [bsN][bsN]bool

	myGrid  [bsN][bsN]bool
	myIdx   [bsN][bsN]int
	myFleet []*bsShipState
	myHit   [bsN][bsN]bool

	botTargets []bsPoint
}

var (
	bsMu   sync.Mutex
	bsInst *battleshipGame
)

func getBattleship() *battleshipGame {
	bsMu.Lock()
	defer bsMu.Unlock()
	if bsInst == nil {
		bsInst = newBattleshipGame()
	}
	return bsInst
}

func newBattleshipGame() *battleshipGame {
	g := &battleshipGame{}
	g.humanTurn = true
	g.status = "playing"
	g.difficulty = diffMedium
	g.message = "Bắn vào bàn đối phương."
	g.enemyGrid, g.enemyIdx, g.enemyFleet = placeFleetRandom()
	g.myGrid, g.myIdx, g.myFleet = placeFleetRandom()
	return g
}

func canPlaceShip(grid *[bsN][bsN]bool, r, c, size int, horiz bool) bool {
	for i := -1; i <= size; i++ {
		for j := -1; j <= 1; j++ {
			var rr, cc int
			if horiz {
				rr, cc = r+j, c+i
			} else {
				rr, cc = r+i, c+j
			}
			if rr < 0 || rr >= bsN || cc < 0 || cc >= bsN {
				continue
			}
			if grid[rr][cc] {
				return false
			}
		}
	}
	return true
}

func placeFleetRandom() ([bsN][bsN]bool, [bsN][bsN]int, []*bsShipState) {
	var grid [bsN][bsN]bool
	var idx [bsN][bsN]int
	for r := 0; r < bsN; r++ {
		for c := 0; c < bsN; c++ {
			idx[r][c] = -1
		}
	}
	var fleet []*bsShipState
	for _, size := range bsShipSizes {
		for {
			horiz := rand.Intn(2) == 0
			var r, c int
			if horiz {
				r = rand.Intn(bsN)
				c = rand.Intn(bsN - size + 1)
			} else {
				r = rand.Intn(bsN - size + 1)
				c = rand.Intn(bsN)
			}
			if !canPlaceShip(&grid, r, c, size, horiz) {
				continue
			}
			ship := &bsShipState{}
			for i := 0; i < size; i++ {
				rr, cc := r, c
				if horiz {
					cc += i
				} else {
					rr += i
				}
				grid[rr][cc] = true
				idx[rr][cc] = len(fleet)
				ship.cells = append(ship.cells, [2]int{rr, cc})
			}
			fleet = append(fleet, ship)
			break
		}
	}
	return grid, idx, fleet
}

func fleetRemaining(fleet []*bsShipState) int {
	n := 0
	for _, s := range fleet {
		n += len(s.cells) - s.hits
	}
	return n
}

func fleetSunkCount(fleet []*bsShipState) int {
	n := 0
	for _, s := range fleet {
		if s.hits >= len(s.cells) {
			n++
		}
	}
	return n
}

// fireAt marks a shot against idx/fleet at (r,c); hit is true iff a ship
// occupies that cell, sunk is true iff that ship's cells are all hit now.
func (g *battleshipGame) fireAt(idx *[bsN][bsN]int, fleet []*bsShipState, r, c int) (hit, sunk bool) {
	i := idx[r][c]
	if i < 0 {
		return false, false
	}
	fleet[i].hits++
	hit = true
	sunk = fleet[i].hits >= len(fleet[i].cells)
	return
}

func (g *battleshipGame) botFireLocked() (u string, hit, sunk bool) {
	r, c, found := -1, -1, false
	for len(g.botTargets) > 0 {
		p := g.botTargets[0]
		g.botTargets = g.botTargets[1:]
		if p.r >= 0 && p.r < bsN && p.c >= 0 && p.c < bsN && !g.myHit[p.r][p.c] {
			r, c, found = p.r, p.c, true
			break
		}
	}
	if !found {
		for {
			rr, cc := rand.Intn(bsN), rand.Intn(bsN)
			if !g.myHit[rr][cc] {
				r, c = rr, cc
				break
			}
		}
	}
	g.myHit[r][c] = true
	hit, sunk = g.fireAt(&g.myIdx, g.myFleet, r, c)
	u = miniSqName(c, r)
	if hit && !sunk {
		g.botTargets = append(g.botTargets, bsPoint{r - 1, c}, bsPoint{r + 1, c}, bsPoint{r, c - 1}, bsPoint{r, c + 1})
	}
	return u, hit, sunk
}

func (g *battleshipGame) enemyBoardRows() []string {
	rows := make([]string, bsN)
	for r := 0; r < bsN; r++ {
		var b strings.Builder
		for c := 0; c < bsN; c++ {
			switch {
			case !g.enemyHit[r][c]:
				b.WriteByte('.')
			case g.enemyIdx[r][c] < 0:
				b.WriteByte('X')
			case g.enemyFleet[g.enemyIdx[r][c]].hits >= len(g.enemyFleet[g.enemyIdx[r][c]].cells):
				b.WriteByte('S')
			default:
				b.WriteByte('H')
			}
		}
		rows[r] = b.String()
	}
	return rows
}

func (g *battleshipGame) myBoardRows() []string {
	rows := make([]string, bsN)
	for r := 0; r < bsN; r++ {
		var b strings.Builder
		for c := 0; c < bsN; c++ {
			switch {
			case g.myHit[r][c] && g.myIdx[r][c] >= 0 && g.myFleet[g.myIdx[r][c]].hits >= len(g.myFleet[g.myIdx[r][c]].cells):
				b.WriteByte('S')
			case g.myHit[r][c] && g.myIdx[r][c] >= 0:
				b.WriteByte('H')
			case g.myHit[r][c]:
				b.WriteByte('X')
			case g.myGrid[r][c]:
				b.WriteByte('O')
			default:
				b.WriteByte('.')
			}
		}
		rows[r] = b.String()
	}
	return rows
}

func (g *battleshipGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *battleshipGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"board": g.enemyBoardRows(), "myBoard": g.myBoardRows(),
		"boardW": bsN, "boardH": bsN,
		"enemyRemaining": fleetRemaining(g.enemyFleet), "myRemaining": fleetRemaining(g.myFleet),
		"enemySunk": fleetSunkCount(g.enemyFleet), "mySunk": fleetSunkCount(g.myFleet),
		"fleetSizes": bsShipSizes,
	}
	return g.baseSnap("battleship", "battleship", extra)
}

func (g *battleshipGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Battleship 8x8 trên web Vector. Ô tàu còn lại của bạn: %d, đối phương: %d. Trạng thái: %s.",
		fleetRemaining(g.myFleet), fleetRemaining(g.enemyFleet), g.status)
}

func (g *battleshipGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newBattleshipGame()
	ng.difficulty = prev
	bsMu.Lock()
	bsInst = ng
	bsMu.Unlock()
	return ng.snapshot()
}

func (g *battleshipGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *battleshipGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *battleshipGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.botThinking || !g.humanTurn {
		return nil
	}
	var out []string
	for r := 0; r < bsN; r++ {
		for c := 0; c < bsN; c++ {
			if !g.enemyHit[r][c] {
				out = append(out, miniSqName(c, r))
			}
		}
	}
	return out
}

func (g *battleshipGame) playUCI(uci string) (map[string]interface{}, error) {
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
	if !ok || file < 0 || file >= bsN || rank < 0 || rank >= bsN {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	if g.enemyHit[rank][file] {
		g.mu.Unlock()
		return nil, fmt.Errorf("already fired there")
	}
	u := miniSqName(file, rank)
	g.enemyHit[rank][file] = true
	hit, sunk := g.fireAt(&g.enemyIdx, g.enemyFleet, rank, file)
	var msg string
	switch {
	case sunk:
		msg = "Bắn chìm tàu!"
	case hit:
		msg = "Trúng!"
	default:
		msg = "Trượt."
	}
	mark := ""
	if hit {
		mark = "*"
	}
	g.lastMove = u
	g.history = append(g.history, "H:"+u+mark)
	g.moves++
	if fleetRemaining(g.enemyFleet) <= 0 {
		g.status = "win"
		g.winner = "human"
		g.message = "Bạn đã đánh chìm toàn bộ hạm đội!"
	} else {
		g.humanTurn = false
		g.message = msg + " Đến lượt bot."
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
	resp["hit"] = hit
	resp["sunk"] = sunk
	g.mu.Unlock()

	statusStr, _ := resp["status"].(string)
	winnerStr, _ := resp["winner"].(string)
	queueGameSpeak("battleship", buildPlaceSpokenHumanOnly("battleship", u, statusStr, winnerStr), "Battleship. Bạn bắn "+u+". "+msg)
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

func (g *battleshipGame) runBotThink(gen int) {
	time.Sleep(600 * time.Millisecond)
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	var botMove, msg string
	if g.status == "playing" && !g.humanTurn {
		u, hit, sunk := g.botFireLocked()
		botMove = u
		switch {
		case sunk:
			msg = "Bot bắn chìm tàu của bạn tại " + u + "!"
		case hit:
			msg = "Bot bắn trúng " + u + "."
		default:
			msg = "Bot bắn trượt " + u + "."
		}
		mark := ""
		if hit {
			mark = "*"
		}
		g.lastMove = u
		g.history = append(g.history, "B:"+u+mark)
		g.moves++
		if fleetRemaining(g.myFleet) <= 0 {
			g.status = "lose"
			g.winner = "bot"
			g.message = "Bot đã đánh chìm hạm đội của bạn."
		} else {
			g.humanTurn = true
			g.message = msg + " Đến lượt bạn."
		}
	}
	g.botThinking = false
	status, winner := g.status, g.winner
	g.mu.Unlock()
	if botMove != "" {
		queueGameSpeak("battleship", buildPlaceSpokenBotOnly("battleship", botMove, status, winner), "Battleship. "+msg)
	}
}

// NewBattleship wires the battleship mod using the generic board-game HTTP surface.
func NewBattleship() *genericBoardMod {
	return newGenericBoardMod("Battleship", "battleship",
		"Battleship (bắn tàu) 8x8 web game — Vector comments; Xiaozhi MCP",
		func() string { return getBattleship().summaryText() },
		func() map[string]interface{} { return getBattleship().snapshot() },
		func() map[string]interface{} { return getBattleship().reset() },
		func(s string) string { return getBattleship().setDifficulty(s) },
		func() string { return getBattleship().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getBattleship().playUCI(u) },
		func() []string { return getBattleship().legalUCIs() },
		func() (string, string) {
			if getChessCommentMode() == chessModeGoogleVI {
				return "Ván battleship mới. Bắn vào bàn đối phương.", "Battleship. Ván mới."
			}
			return "New battleship game. Fire at the enemy board.", "Battleship. Ván mới."
		},
	)
}
