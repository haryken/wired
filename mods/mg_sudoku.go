package mods

import (
	"fmt"
	"strings"
	"sync"
)

// Sudoku: 9x9, solo puzzle (no bot turn). UCI "a0:5" / "a05" sets digit 1-9;
// "a0:0" / "a0x" clears; "hint" fills one correct empty cell.

const sudokuN = 9

// sudokuPuzzles holds one hard-coded, uniquely-solvable puzzle per difficulty,
// as an 81-char string (row-major, '0' = empty).
var sudokuPuzzles = map[string]string{
	diffEasy:   "000068710300970804081002630512703090800629100600154000067005302420000900138000540",
	diffMedium: "009060015000000800001000630502700400800009100603054278907400300400830000100096000",
	diffHard:   "200060000006001804701500609000783400000000000600004070000005300000800060100096007",
}

type sudokuGame struct {
	miniCommon
	board [sudokuN][sudokuN]int
	given [sudokuN][sudokuN]bool
}

var (
	sudokuMu   sync.Mutex
	sudokuInst *sudokuGame
)

func getSudoku() *sudokuGame {
	sudokuMu.Lock()
	defer sudokuMu.Unlock()
	if sudokuInst == nil {
		sudokuInst = newSudokuGame(diffMedium)
	}
	return sudokuInst
}

func newSudokuGame(level string) *sudokuGame {
	g := &sudokuGame{}
	g.difficulty = normalizeGameDifficulty(level)
	g.humanTurn = true
	g.status = "playing"
	g.message = "Điền số 1-9. Không được lặp trong hàng, cột, hoặc ô 3x3."
	g.loadPuzzleLocked(g.difficulty)
	return g
}

func (g *sudokuGame) loadPuzzleLocked(level string) {
	s := sudokuPuzzles[normalizeGameDifficulty(level)]
	if len(s) != sudokuN*sudokuN {
		s = sudokuPuzzles[diffMedium]
	}
	for i, ch := range s {
		r, c := i/sudokuN, i%sudokuN
		if ch >= '1' && ch <= '9' {
			g.board[r][c] = int(ch - '0')
			g.given[r][c] = true
		} else {
			g.board[r][c] = 0
			g.given[r][c] = false
		}
	}
}

func parseSudokuUCI(s string) (file, rank, val int, hint, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "hint" {
		return 0, 0, 0, true, true
	}
	if len(s) < 2 {
		return 0, 0, 0, false, false
	}
	if s[0] < 'a' || s[0] > 'i' {
		return 0, 0, 0, false, false
	}
	if s[1] < '0' || s[1] > '8' {
		return 0, 0, 0, false, false
	}
	file = int(s[0] - 'a')
	rank = int(s[1] - '0')
	rest := strings.TrimPrefix(s[2:], ":")
	switch rest {
	case "":
		return 0, 0, 0, false, false
	case "x", "0":
		val = 0
	default:
		if len(rest) != 1 || rest[0] < '1' || rest[0] > '9' {
			return 0, 0, 0, false, false
		}
		val = int(rest[0] - '0')
	}
	return file, rank, val, false, true
}

func sudokuValidPlace(board *[sudokuN][sudokuN]int, r, c, v int) bool {
	for i := 0; i < sudokuN; i++ {
		if board[r][i] == v || board[i][c] == v {
			return false
		}
	}
	br, bc := (r/3)*3, (c/3)*3
	for i := br; i < br+3; i++ {
		for j := bc; j < bc+3; j++ {
			if board[i][j] == v {
				return false
			}
		}
	}
	return true
}

// sudokuSolve fills empty (0) cells of board in place via backtracking.
func sudokuSolve(board *[sudokuN][sudokuN]int) bool {
	r, c, found := -1, -1, false
	for i := 0; i < sudokuN && !found; i++ {
		for j := 0; j < sudokuN; j++ {
			if board[i][j] == 0 {
				r, c, found = i, j, true
				break
			}
		}
	}
	if !found {
		return true
	}
	for v := 1; v <= 9; v++ {
		if sudokuValidPlace(board, r, c, v) {
			board[r][c] = v
			if sudokuSolve(board) {
				return true
			}
			board[r][c] = 0
		}
	}
	return false
}

// hasConflictAt reports whether the digit at (r,c) duplicates in its row/col/box.
func (g *sudokuGame) hasConflictAt(r, c int) bool {
	v := g.board[r][c]
	if v == 0 {
		return false
	}
	for i := 0; i < sudokuN; i++ {
		if i != c && g.board[r][i] == v {
			return true
		}
		if i != r && g.board[i][c] == v {
			return true
		}
	}
	br, bc := (r/3)*3, (c/3)*3
	for i := br; i < br+3; i++ {
		for j := bc; j < bc+3; j++ {
			if (i != r || j != c) && g.board[i][j] == v {
				return true
			}
		}
	}
	return false
}

func (g *sudokuGame) boardValid() bool {
	for r := 0; r < sudokuN; r++ {
		for c := 0; c < sudokuN; c++ {
			if g.hasConflictAt(r, c) {
				return false
			}
		}
	}
	return true
}

func (g *sudokuGame) boardFull() bool {
	for r := 0; r < sudokuN; r++ {
		for c := 0; c < sudokuN; c++ {
			if g.board[r][c] == 0 {
				return false
			}
		}
	}
	return true
}

// findHintLocked solves the original puzzle (fixed givens only) and returns a
// correct value for the first empty cell of the live board; that cell then
// becomes fixed. Must hold lock.
func (g *sudokuGame) findHintLocked() (file, rank, val int, found bool) {
	var puzzle [sudokuN][sudokuN]int
	for r := 0; r < sudokuN; r++ {
		for c := 0; c < sudokuN; c++ {
			if g.given[r][c] {
				puzzle[r][c] = g.board[r][c]
			}
		}
	}
	if !sudokuSolve(&puzzle) {
		return 0, 0, 0, false
	}
	for r := 0; r < sudokuN; r++ {
		for c := 0; c < sudokuN; c++ {
			if g.board[r][c] == 0 {
				g.board[r][c] = puzzle[r][c]
				g.given[r][c] = true
				return c, r, puzzle[r][c], true
			}
		}
	}
	return 0, 0, 0, false
}

func (g *sudokuGame) boardRows() []string {
	rows := make([]string, sudokuN)
	for r := 0; r < sudokuN; r++ {
		var b strings.Builder
		for c := 0; c < sudokuN; c++ {
			if g.board[r][c] == 0 {
				b.WriteByte('.')
			} else {
				b.WriteByte(byte('0' + g.board[r][c]))
			}
		}
		rows[r] = b.String()
	}
	return rows
}

func (g *sudokuGame) givenRows() []string {
	rows := make([]string, sudokuN)
	for r := 0; r < sudokuN; r++ {
		var b strings.Builder
		for c := 0; c < sudokuN; c++ {
			if g.given[r][c] {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		rows[r] = b.String()
	}
	return rows
}

func (g *sudokuGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *sudokuGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"board": g.boardRows(), "given": g.givenRows(),
		"boardW": sudokuN, "boardH": sudokuN,
		"valid": g.boardValid(),
	}
	return g.baseSnap("sudoku", "sudoku", extra)
}

func (g *sudokuGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Sudoku 9x9 độ khó %s trên web Vector. Trạng thái: %s. ", normalizeGameDifficulty(g.difficulty), g.status))
	if g.lastMove != "" {
		b.WriteString("Nước gần nhất: " + g.lastMove + ". ")
	}
	b.WriteString(fmt.Sprintf("Số nước: %d.", g.moves))
	return b.String()
}

func (g *sudokuGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newSudokuGame(prev)
	sudokuMu.Lock()
	sudokuInst = ng
	sudokuMu.Unlock()
	return ng.snapshot()
}

func (g *sudokuGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *sudokuGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *sudokuGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" {
		return nil
	}
	var out []string
	for r := 0; r < sudokuN; r++ {
		for c := 0; c < sudokuN; c++ {
			if !g.given[r][c] {
				out = append(out, miniSqName(c, r))
			}
		}
	}
	return out
}

func (g *sudokuGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	file, rank, val, hint, ok := parseSudokuUCI(uci)
	if !ok {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	var moveDesc string
	if hint {
		f, r, v, found := g.findHintLocked()
		if !found {
			g.mu.Unlock()
			return nil, fmt.Errorf("no hint available")
		}
		u := miniSqName(f, r)
		g.lastMove = u + ":" + fmt.Sprintf("%d", v)
		g.history = append(g.history, "hint:"+u)
		g.moves++
		moveDesc = "gợi ý " + u + "=" + fmt.Sprintf("%d", v)
	} else {
		if rank < 0 || rank >= sudokuN || file < 0 || file >= sudokuN {
			g.mu.Unlock()
			return nil, fmt.Errorf("out of range")
		}
		if g.given[rank][file] {
			g.mu.Unlock()
			return nil, fmt.Errorf("cell is fixed")
		}
		u := miniSqName(file, rank)
		g.board[rank][file] = val
		if val == 0 {
			g.lastMove = u + ":x"
			g.history = append(g.history, "clear:"+u)
			moveDesc = "xoá " + u
		} else {
			g.lastMove = u + ":" + fmt.Sprintf("%d", val)
			g.history = append(g.history, fmt.Sprintf("%s:%d", u, val))
			moveDesc = fmt.Sprintf("%s=%d", u, val)
		}
		g.moves++
	}
	valid := g.boardValid()
	full := g.boardFull()
	if full && valid {
		g.status = "win"
		g.winner = "human"
		g.message = "Chúc mừng! Bạn đã giải xong Sudoku."
	} else if !valid {
		g.message = "Có xung đột trong hàng, cột, hoặc ô 3x3."
	} else {
		g.message = "Tiếp tục điền số."
	}
	resp := g.snapshotLocked()
	resp["youMove"] = moveDesc
	g.mu.Unlock()

	statusStr, _ := resp["status"].(string)
	winnerStr, _ := resp["winner"].(string)
	queueGameSpeak("sudoku", buildPlaceSpokenHumanOnly("sudoku", moveDesc, statusStr, winnerStr), "Sudoku. "+moveDesc+".")
	return resp, nil
}

// NewSudoku wires the sudoku mod using the generic board-game HTTP surface.
func NewSudoku() *genericBoardMod {
	return newGenericBoardMod("Sudoku", "sudoku",
		"Sudoku 9x9 web game — Vector comments; Xiaozhi MCP",
		func() string { return getSudoku().summaryText() },
		func() map[string]interface{} { return getSudoku().snapshot() },
		func() map[string]interface{} { return getSudoku().reset() },
		func(s string) string { return getSudoku().setDifficulty(s) },
		func() string { return getSudoku().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getSudoku().playUCI(u) },
		func() []string { return getSudoku().legalUCIs() },
		func() (string, string) {
			if getChessCommentMode() == chessModeGoogleVI {
				return "Ván sudoku mới. Điền số từ 1 đến 9.", "Sudoku. Ván mới."
			}
			return "New sudoku puzzle. Fill in digits 1 to 9.", "Sudoku. Ván mới."
		},
	)
}
