package mods

import "testing"

func placeBestO(b *[caroN][caroN]caroCell) (f, r, sc int) {
	bestS := -1 << 30
	bestF, bestR := -1, -1
	for y := 0; y < caroN; y++ {
		for x := 0; x < caroN; x++ {
			if b[y][x] != caroEmpty {
				continue
			}
			s := caroEvalPlace(b, x, y, caroO)
			if s > bestS {
				bestS, bestF, bestR = s, x, y
			}
		}
	}
	return bestF, bestR, bestS
}

func TestCaroBlocksOpenThreeWhenNoOwnForce(t *testing.T) {
	var b [caroN][caroN]caroCell
	// Vertical open-3 for X at i7,i8,i9
	b[7][8], b[8][8], b[9][8] = caroX, caroX, caroX
	// Bot dead-4 on rank 8: e–h blocked by d8 + i8
	b[8][3] = caroX
	b[8][4], b[8][5], b[8][6], b[8][7] = caroO, caroO, caroO, caroO
	// do NOT give O a live-3/4 of its own (f7/f9 would create force)

	f, r, sc := placeBestO(&b)
	// Must block i6 or i10
	if !(f == 8 && (r == 6 || r == 10)) {
		t.Fatalf("expected block i6/i10, got %c%d sc=%d", 'a'+f, r, sc)
	}
}

func TestCaroOwnLiveFourBeatsOpenThreeBlock(t *testing.T) {
	var b [caroN][caroN]caroCell
	b[7][8], b[8][8], b[9][8] = caroX, caroX, caroX
	// O vertical live three at f7-f9 + open ends
	b[7][5], b[8][5], b[9][5] = caroO, caroO, caroO

	f, r, sc := placeBestO(&b)
	// f6 or f10 live four preferred over i-block
	if !(f == 5 && (r == 6 || r == 10)) {
		t.Fatalf("expected own live-four f6/f10, got %c%d sc=%d", 'a'+f, r, sc)
	}
}

func TestCaroBlocksOpenFourWin(t *testing.T) {
	var b [caroN][caroN]caroCell
	// X half-open four needing one to win: a0-a3, open a4
	b[0][0], b[1][0], b[2][0], b[3][0] = caroX, caroX, caroX, caroX
	f, r, sc := placeBestO(&b)
	if f != 0 || r != 4 {
		t.Fatalf("expected block a4, got %c%d sc=%d", 'a'+f, r, sc)
	}
	if sc < 400_000 {
		t.Fatalf("block-win score too low: %d", sc)
	}
}

func TestCaroTakesWin(t *testing.T) {
	var b [caroN][caroN]caroCell
	b[8][4], b[8][5], b[8][6], b[8][7] = caroO, caroO, caroO, caroO
	// e8 wins
	f, r, sc := placeBestO(&b)
	if !(f == 3 || f == 8) || r != 8 {
		// e=3 or i=8 adjacent
		// actually open both ends d and i: f=4..7, place f=3 (e) or f=8 (i)
		if !((f == 3 || f == 8) && r == 8) {
			t.Fatalf("expected win e8/i8, got %c%d sc=%d", 'a'+f, r, sc)
		}
	}
	if sc < 900_000 {
		t.Fatalf("win score too low: %d", sc)
	}
}
