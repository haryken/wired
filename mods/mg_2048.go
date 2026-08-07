package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// Classic 2048, 4x4, solo. UCI u/d/l/r (or up/down/left/right).

const g2048N = 4

type g2048Game struct {
	miniCommon
	board [g2048N][g2048N]int
	score int
	won   bool
}

var (
	g2048Mu   sync.Mutex
	g2048Inst *g2048Game
)

func getG2048() *g2048Game {
	g2048Mu.Lock()
	defer g2048Mu.Unlock()
	if g2048Inst == nil {
		g2048Inst = newG2048Game()
	}
	return g2048Inst
}

func newG2048Game() *g2048Game {
	g := &g2048Game{}
	g.humanTurn = true
	g.status = "playing"
	g.difficulty = diffMedium
	g.message = "Gõ u/d/l/r để trượt các ô."
	g.spawnTile()
	g.spawnTile()
	return g
}

func (g *g2048Game) spawnTile() bool {
	var empties [][2]int
	for r := 0; r < g2048N; r++ {
		for c := 0; c < g2048N; c++ {
			if g.board[r][c] == 0 {
				empties = append(empties, [2]int{r, c})
			}
		}
	}
	if len(empties) == 0 {
		return false
	}
	p := empties[rand.Intn(len(empties))]
	v := 2
	if rand.Intn(10) == 0 {
		v = 4
	}
	g.board[p[0]][p[1]] = v
	return true
}

func (g *g2048Game) hasTile(v int) bool {
	for r := 0; r < g2048N; r++ {
		for c := 0; c < g2048N; c++ {
			if g.board[r][c] == v {
				return true
			}
		}
	}
	return false
}

func (g *g2048Game) canMove() bool {
	for r := 0; r < g2048N; r++ {
		for c := 0; c < g2048N; c++ {
			if g.board[r][c] == 0 {
				return true
			}
			if c+1 < g2048N && g.board[r][c] == g.board[r][c+1] {
				return true
			}
			if r+1 < g2048N && g.board[r][c] == g.board[r+1][c] {
				return true
			}
		}
	}
	return false
}

func reverseRow(row [g2048N]int) [g2048N]int {
	var out [g2048N]int
	for i := 0; i < g2048N; i++ {
		out[i] = row[g2048N-1-i]
	}
	return out
}

func getCol(board [g2048N][g2048N]int, c int) [g2048N]int {
	var out [g2048N]int
	for r := 0; r < g2048N; r++ {
		out[r] = board[r][c]
	}
	return out
}

func setCol(board *[g2048N][g2048N]int, c int, col [g2048N]int) {
	for r := 0; r < g2048N; r++ {
		board[r][c] = col[r]
	}
}

// g2048SlideRowLeft compacts+merges a single row to the left.
func g2048SlideRowLeft(row [g2048N]int) (newRow [g2048N]int, gained int, moved bool) {
	var vals []int
	for _, v := range row {
		if v != 0 {
			vals = append(vals, v)
		}
	}
	var merged []int
	skip := false
	for i := 0; i < len(vals); i++ {
		if skip {
			skip = false
			continue
		}
		if i+1 < len(vals) && vals[i] == vals[i+1] {
			m := vals[i] * 2
			merged = append(merged, m)
			gained += m
			skip = true
		} else {
			merged = append(merged, vals[i])
		}
	}
	for len(merged) < g2048N {
		merged = append(merged, 0)
	}
	for i := 0; i < g2048N; i++ {
		newRow[i] = merged[i]
	}
	moved = newRow != row
	return
}

// slideBoard applies one slide direction to a board copy, without mutating the caller's board.
func slideBoard(board [g2048N][g2048N]int, dir string) ([g2048N][g2048N]int, int, bool) {
	moved := false
	gained := 0
	switch dir {
	case "l":
		for r := 0; r < g2048N; r++ {
			nr, gn, mv := g2048SlideRowLeft(board[r])
			if mv {
				moved = true
			}
			gained += gn
			board[r] = nr
		}
	case "r":
		for r := 0; r < g2048N; r++ {
			rev := reverseRow(board[r])
			nr, gn, mv := g2048SlideRowLeft(rev)
			if mv {
				moved = true
			}
			gained += gn
			board[r] = reverseRow(nr)
		}
	case "u":
		for c := 0; c < g2048N; c++ {
			col := getCol(board, c)
			nc, gn, mv := g2048SlideRowLeft(col)
			if mv {
				moved = true
			}
			gained += gn
			setCol(&board, c, nc)
		}
	case "d":
		for c := 0; c < g2048N; c++ {
			col := reverseRow(getCol(board, c))
			nc, gn, mv := g2048SlideRowLeft(col)
			if mv {
				moved = true
			}
			gained += gn
			setCol(&board, c, reverseRow(nc))
		}
	}
	return board, gained, moved
}

func parse2048Dir(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "u", "up":
		return "u", true
	case "d", "down":
		return "d", true
	case "l", "left":
		return "l", true
	case "r", "right":
		return "r", true
	default:
		return "", false
	}
}

func (g *g2048Game) boardRows() []string {
	rows := make([]string, g2048N)
	for r := 0; r < g2048N; r++ {
		cells := make([]string, g2048N)
		for c := 0; c < g2048N; c++ {
			if g.board[r][c] == 0 {
				cells[c] = "."
			} else {
				cells[c] = fmt.Sprintf("%d", g.board[r][c])
			}
		}
		rows[r] = strings.Join(cells, " ")
	}
	return rows
}

func (g *g2048Game) tiles() [][]int {
	out := make([][]int, g2048N)
	for r := 0; r < g2048N; r++ {
		row := make([]int, g2048N)
		copy(row, g.board[r][:])
		out[r] = row
	}
	return out
}

func (g *g2048Game) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *g2048Game) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"board": g.boardRows(), "tiles": g.tiles(),
		"boardW": g2048N, "boardH": g2048N, "score": g.score,
	}
	return g.baseSnap("g2048", "tiles2048", extra)
}

func (g *g2048Game) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("2048 (4x4) trên web Vector. Điểm: %d. Trạng thái: %s.", g.score, g.status)
}

func (g *g2048Game) reset() map[string]interface{} {
	ng := newG2048Game()
	g2048Mu.Lock()
	g2048Inst = ng
	g2048Mu.Unlock()
	return ng.snapshot()
}

func (g *g2048Game) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *g2048Game) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *g2048Game) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status == "lose" {
		return nil
	}
	var out []string
	for _, d := range []string{"u", "d", "l", "r"} {
		_, _, moved := slideBoard(g.board, d)
		if moved {
			out = append(out, d)
		}
	}
	return out
}

func (g *g2048Game) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status == "lose" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	dir, ok := parse2048Dir(uci)
	if !ok {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	nb, gained, moved := slideBoard(g.board, dir)
	if !moved {
		g.mu.Unlock()
		return nil, fmt.Errorf("no tiles moved")
	}
	g.board = nb
	g.score += gained
	g.moves++
	g.lastMove = dir
	g.history = append(g.history, dir)
	g.spawnTile()
	if !g.won && g.hasTile(2048) {
		g.won = true
		g.status = "win"
		g.winner = "human"
		g.message = "Bạn đạt 2048!"
	}
	if !g.canMove() {
		g.status = "lose"
		g.message = fmt.Sprintf("Hết nước đi. Điểm: %d.", g.score)
	} else if g.status == "playing" {
		g.message = "Tiếp tục."
	}
	scoreCopy := g.score
	resp := g.snapshotLocked()
	resp["youMove"] = dir
	resp["gained"] = gained
	g.mu.Unlock()

	statusStr, _ := resp["status"].(string)
	winnerStr, _ := resp["winner"].(string)
	queueGameSpeak("g2048", buildPlaceSpokenHumanOnly("g2048", dir, statusStr, winnerStr), fmt.Sprintf("2048. Điểm: %d.", scoreCopy))
	return resp, nil
}

// NewG2048 wires the 2048 mod using the generic board-game HTTP surface.
func NewG2048() *genericBoardMod {
	return newGenericBoardMod("G2048", "g2048",
		"2048 puzzle web game — Vector comments; Xiaozhi MCP",
		func() string { return getG2048().summaryText() },
		func() map[string]interface{} { return getG2048().snapshot() },
		func() map[string]interface{} { return getG2048().reset() },
		func(s string) string { return getG2048().setDifficulty(s) },
		func() string { return getG2048().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getG2048().playUCI(u) },
		func() []string { return getG2048().legalUCIs() },
		func() (string, string) {
			if getChessCommentMode() == chessModeGoogleVI {
				return "Ván 2048 mới. Gộp các ô cùng số để đạt 2048.", "2048. Ván mới."
			}
			return "New 2048 game. Merge matching tiles to reach 2048.", "2048. Ván mới."
		},
	)
}
