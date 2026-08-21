package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// Minesweeper: 9x9, mine count depends on difficulty. UCI "a0" reveals;
// "fa0" or "flag:a0" toggles a flag. First click never a mine.

const minesN = 9

type minesGame struct {
	miniCommon
	board         [minesN][minesN]int // -1 mine, 0-8 neighbor count
	revealed      [minesN][minesN]bool
	flagged       [minesN][minesN]bool
	mineCount     int
	started       bool
	revealedCount int
}

var (
	minesMu   sync.Mutex
	minesInst *minesGame
)

func getMinesweeper() *minesGame {
	minesMu.Lock()
	defer minesMu.Unlock()
	if minesInst == nil {
		minesInst = newMinesGame(diffMedium)
	}
	return minesInst
}

func minesCountForDifficulty(d string) int {
	switch normalizeGameDifficulty(d) {
	case diffEasy:
		return 10
	case diffHard:
		return 20
	default:
		return 15
	}
}

func newMinesGame(level string) *minesGame {
	g := &minesGame{}
	g.difficulty = normalizeGameDifficulty(level)
	g.humanTurn = true
	g.status = "playing"
	g.mineCount = minesCountForDifficulty(g.difficulty)
	g.message = "Mở một ô để bắt đầu. Gõ f rồi ô để cắm cờ."
	return g
}

// placeMinesLocked places mines avoiding (er,ec) and its neighbors, then
// computes neighbor counts. Must hold lock.
func (g *minesGame) placeMinesLocked(er, ec int) {
	banned := map[[2]int]bool{}
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			nr, nc := er+dr, ec+dc
			if nr >= 0 && nr < minesN && nc >= 0 && nc < minesN {
				banned[[2]int{nr, nc}] = true
			}
		}
	}
	var cells [][2]int
	for r := 0; r < minesN; r++ {
		for c := 0; c < minesN; c++ {
			if !banned[[2]int{r, c}] {
				cells = append(cells, [2]int{r, c})
			}
		}
	}
	rand.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	n := g.mineCount
	if n > len(cells) {
		n = len(cells)
	}
	for i := 0; i < n; i++ {
		r, c := cells[i][0], cells[i][1]
		g.board[r][c] = -1
	}
	for r := 0; r < minesN; r++ {
		for c := 0; c < minesN; c++ {
			if g.board[r][c] == -1 {
				continue
			}
			cnt := 0
			for dr := -1; dr <= 1; dr++ {
				for dc := -1; dc <= 1; dc++ {
					if dr == 0 && dc == 0 {
						continue
					}
					nr, nc := r+dr, c+dc
					if nr >= 0 && nr < minesN && nc >= 0 && nc < minesN && g.board[nr][nc] == -1 {
						cnt++
					}
				}
			}
			g.board[r][c] = cnt
		}
	}
	g.started = true
}

// revealLocked flood-fills from (r,c). Must hold lock.
func (g *minesGame) revealLocked(r, c int) {
	if r < 0 || r >= minesN || c < 0 || c >= minesN {
		return
	}
	if g.revealed[r][c] || g.flagged[r][c] {
		return
	}
	g.revealed[r][c] = true
	g.revealedCount++
	if g.board[r][c] == 0 {
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				if dr == 0 && dc == 0 {
					continue
				}
				g.revealLocked(r+dr, c+dc)
			}
		}
	}
}

func parseMinesUCI(s string) (file, rank int, flag bool, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(s, "flag:"):
		f, r, o := miniParseSq(s[len("flag:"):])
		return f, r, true, o
	case len(s) >= 3 && s[0] == 'f':
		f, r, o := miniParseSq(s[1:])
		return f, r, true, o
	default:
		f, r, o := miniParseSq(s)
		return f, r, false, o
	}
}

func (g *minesGame) boardRows() []string {
	rows := make([]string, minesN)
	lost := g.status == "lose"
	for r := 0; r < minesN; r++ {
		var b strings.Builder
		for c := 0; c < minesN; c++ {
			switch {
			case g.flagged[r][c] && !g.revealed[r][c]:
				b.WriteByte('F')
			case !g.revealed[r][c]:
				if lost && g.board[r][c] == -1 {
					b.WriteByte('*')
				} else {
					b.WriteByte('.')
				}
			case g.board[r][c] == -1:
				b.WriteByte('*')
			case g.board[r][c] == 0:
				b.WriteByte(' ')
			default:
				b.WriteByte(byte('0' + g.board[r][c]))
			}
		}
		rows[r] = b.String()
	}
	return rows
}

func (g *minesGame) countFlags() int {
	n := 0
	for r := 0; r < minesN; r++ {
		for c := 0; c < minesN; c++ {
			if g.flagged[r][c] {
				n++
			}
		}
	}
	return n
}

func (g *minesGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *minesGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"board": g.boardRows(), "boardW": minesN, "boardH": minesN,
		"mines": g.mineCount, "flagsUsed": g.countFlags(), "revealed": g.revealedCount,
	}
	return g.baseSnap("mines", "mines", extra)
}

func (g *minesGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Dò mìn %dx%d, %d mìn, độ khó %s trên web Vector. Trạng thái: %s.",
		minesN, minesN, g.mineCount, normalizeGameDifficulty(g.difficulty), g.status)
}

func (g *minesGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newMinesGame(prev)
	minesMu.Lock()
	minesInst = ng
	minesMu.Unlock()
	return ng.snapshot()
}

func (g *minesGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *minesGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *minesGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" {
		return nil
	}
	var out []string
	for r := 0; r < minesN; r++ {
		for c := 0; c < minesN; c++ {
			if !g.revealed[r][c] && !g.flagged[r][c] {
				out = append(out, miniSqName(c, r))
			}
		}
	}
	return out
}

func (g *minesGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	file, rank, flag, ok := parseMinesUCI(uci)
	if !ok || file < 0 || file >= minesN || rank < 0 || rank >= minesN {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	u := miniSqName(file, rank)
	if flag {
		if g.revealed[rank][file] {
			g.mu.Unlock()
			return nil, fmt.Errorf("cell already revealed")
		}
		g.flagged[rank][file] = !g.flagged[rank][file]
		g.lastMove = "flag:" + u
		g.history = append(g.history, "flag:"+u)
		g.message = "Đã cắm/gỡ cờ."
		resp := g.snapshotLocked()
		resp["youMove"] = "flag:" + u
		g.mu.Unlock()
		return resp, nil
	}
	if g.flagged[rank][file] {
		g.mu.Unlock()
		return nil, fmt.Errorf("cell flagged")
	}
	if g.revealed[rank][file] {
		g.mu.Unlock()
		return nil, fmt.Errorf("already revealed")
	}
	if !g.started {
		g.placeMinesLocked(rank, file)
	}
	if g.board[rank][file] == -1 {
		g.revealed[rank][file] = true
		g.status = "lose"
		g.winner = "bot"
		g.message = "Bạn đã đạp trúng mìn!"
		g.lastMove = u
		g.history = append(g.history, "boom:"+u)
		for r := 0; r < minesN; r++ {
			for c := 0; c < minesN; c++ {
				if g.board[r][c] == -1 {
					g.revealed[r][c] = true
				}
			}
		}
	} else {
		g.revealLocked(rank, file)
		g.lastMove = u
		g.history = append(g.history, u)
		if g.revealedCount == minesN*minesN-g.mineCount {
			g.status = "win"
			g.winner = "human"
			g.message = "Bạn đã dò hết mìn!"
		} else {
			g.message = "Tiếp tục dò mìn."
		}
	}
	g.moves++
	resp := g.snapshotLocked()
	resp["youMove"] = u
	g.mu.Unlock()

	statusStr, _ := resp["status"].(string)
	winnerStr, _ := resp["winner"].(string)
	queueGameSpeak("mines", buildPlaceSpokenHumanOnly("mines", u, statusStr, winnerStr), "Dò mìn. Ô: "+u+".")
	return resp, nil
}

// NewMinesweeper wires the minesweeper mod using the generic board-game HTTP surface.
func NewMinesweeper() *genericBoardMod {
	return newGenericBoardMod("Minesweeper", "mines",
		"Minesweeper (dò mìn) 9x9 web game — Vector comments; Xiaozhi MCP",
		func() string { return getMinesweeper().summaryText() },
		func() map[string]interface{} { return getMinesweeper().snapshot() },
		func() map[string]interface{} { return getMinesweeper().reset() },
		func(s string) string { return getMinesweeper().setDifficulty(s) },
		func() string { return getMinesweeper().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getMinesweeper().playUCI(u) },
		func() []string { return getMinesweeper().legalUCIs() },
		func() (string, string) {
			if chessPreferVIText() {
				return "Ván dò mìn mới. Mở một ô để bắt đầu.", "Dò mìn. Ván mới."
			}
			return "New minesweeper game. Reveal a cell to start.", "Minesweeper. Ván mới."
		},
	)
}
