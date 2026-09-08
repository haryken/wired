package mods

import "testing"

func TestChessSEEEqualQueenTradeNegative(t *testing.T) {
	g := newChessGame()
	// Clear path d-file partially: standard start Qxd1 not legal for black.
	// Place: white Q d1, black Q d8, empty d-file middle, white king e1 so Qxd1 checks.
	// Force custom board:
	for i := range g.board {
		g.board[i] = empty
	}
	g.board[0] = 'K'  // a1 white king away? Wait put king e1=4
	g.board[4] = 'K'  // e1
	g.board[3] = 'Q'  // d1
	g.board[59] = 'q' // d8 black queen
	g.board[60] = 'k' // e8
	g.white = false

	// Black q d8 -> d1 (rank 7 file 3 -> rank 0 file 3)
	from, to := 59, 3
	m := chessMove{From: from, To: to, Capture: 'Q'}
	// Ensure apply would work: open file - ranks 1-6 on d empty: sq = r*8+3
	see := g.chessSEE(m)
	if see >= 0 {
		t.Fatalf("equal QxQ into recapture should be SEE < 0, got %d", see)
	}
}

func TestChessHardAvoidsQueenHang(t *testing.T) {
	g := newChessGame()
	for i := range g.board {
		g.board[i] = empty
	}
	g.board[4] = 'K'
	g.board[3] = 'Q'
	g.board[59] = 'q'
	g.board[60] = 'k'
	// add a quiet developing knight for black to pick instead
	g.board[57] = 'n' // b8
	g.board[56] = 'r'
	g.board[58] = 'b'
	g.board[61] = 'b'
	g.board[62] = 'n'
	g.board[63] = 'r'
	// white pawns to block nothing on d-file
	g.white = false
	g.castle = ""
	g.ep = -1

	legal := g.legalMoves()
	if len(legal) == 0 {
		t.Fatal("no legal moves")
	}
	// Find Qxd1 if present
	hasQx := false
	for _, m := range legal {
		if m.From == 59 && m.To == 3 {
			hasQx = true
		}
	}
	if !hasQx {
		t.Fatalf("expected Qxd1 legal, got %d moves", len(legal))
	}
	best := g.pickChessHard(legal)
	if best.From == 59 && best.To == 3 {
		t.Fatalf("hard picked hung QxQ SEE trade: %v", moveUCI(best))
	}
}
