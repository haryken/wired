package mods

import "testing"

// Squares: sq = rank*8 + file, a1 = 0, h8 = 63; white pieces are uppercase.

func TestChessLegalMovesKeepsState(t *testing.T) {
	g := newChessGame()
	g.halfmove = 5
	_ = g.legalMoves()
	if g.castle != "KQkq" {
		t.Fatalf("castle rights lost: %q", g.castle)
	}
	if g.ep != -1 {
		t.Fatalf("ep corrupted: %d", g.ep)
	}
	if g.halfmove != 5 {
		t.Fatalf("halfmove corrupted: %d", g.halfmove)
	}
}

func TestChessCastlingAfterScan(t *testing.T) {
	g := newChessGame()
	g.board[5] = empty // f1
	g.board[6] = empty // g1
	_ = g.legalMoves()
	found := false
	for _, m := range g.legalMoves() {
		if m.From == 4 && m.To == 6 && m.IsCastle {
			found = true
		}
	}
	if !found {
		t.Fatal("white O-O should be legal")
	}
}

func TestChessEnPassantAfterScan(t *testing.T) {
	g := newChessGame()
	g.board[12] = empty // e2
	g.board[36] = 'P'   // e5
	g.white = false
	var d7d5 *chessMove
	for _, m := range g.legalMoves() {
		if m.From == 51 && m.To == 35 {
			mm := m
			d7d5 = &mm
		}
	}
	if d7d5 == nil {
		t.Fatal("d7d5 not found")
	}
	g.apply(d7d5)
	_ = g.legalMoves()
	found := false
	for _, m := range g.legalMoves() {
		if m.From == 36 && m.To == 43 && m.IsEnPassant {
			found = true
		}
	}
	if !found {
		t.Fatal("e5xd6 en passant should be legal")
	}
}

func TestChessNoPhantomA1EnPassant(t *testing.T) {
	g := newChessGame()
	for i := range g.board {
		g.board[i] = empty
	}
	g.board[4] = 'K'  // e1
	g.board[60] = 'k' // e8
	g.board[9] = 'p'  // b2
	g.board[8] = 'P'  // a2
	g.white = false
	g.castle = ""
	g.ep = -1
	_ = g.legalMoves()
	for _, m := range g.legalMoves() {
		if m.From == 9 && m.To == 0 && m.IsEnPassant {
			t.Fatal("phantom en passant b2->a1")
		}
	}
}
