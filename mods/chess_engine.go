package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Compact mailbox chess for web :80 + Xiaozhi MCP (P0).
// Human = white; bot = black after each legal human move.

type chessPiece byte

const (
	empty chessPiece = 0
)

func isWhite(p chessPiece) bool { return p >= 'A' && p <= 'Z' }
func isBlack(p chessPiece) bool { return p >= 'a' && p <= 'z' }
func isEnemy(p, side chessPiece) bool {
	if p == empty {
		return false
	}
	if isWhite(side) {
		return isBlack(p)
	}
	return isWhite(p)
}
func isFriend(p, side chessPiece) bool {
	if p == empty {
		return false
	}
	if isWhite(side) {
		return isWhite(p)
	}
	return isBlack(p)
}
func toLower(p chessPiece) chessPiece {
	if p >= 'A' && p <= 'Z' {
		return p + 32
	}
	return p
}

type chessMove struct {
	From, To     int
	Promo        chessPiece // 0 or Q/q/R/r/...
	Capture      chessPiece
	IsCastle     bool
	IsEnPassant  bool
	PrevEP       int
	PrevCastle   string
	PrevHalfmove int
}

type chessGame struct {
	mu          sync.Mutex
	board       [64]chessPiece
	white       bool
	castle      string // KQkq subset
	ep          int    // -1 none
	halfmove    int
	fullmove    int
	history     []string // UCI moves
	lastUCI     string
	lastSAN     string
	status      string // playing | check | checkmate | stalemate
	winner      string // white | black | ""
	message     string
	difficulty  string // easy | medium | hard
	botThinking bool
	thinkGen    int // invalidate in-flight bot thinks on reset / new move
}

var (
	chessMu   sync.Mutex
	chessInst *chessGame
	chessRNG  = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func getChess() *chessGame {
	chessMu.Lock()
	defer chessMu.Unlock()
	if chessInst == nil {
		chessInst = newChessGame()
	}
	return chessInst
}

func newChessGame() *chessGame {
	g := &chessGame{
		white:      true,
		castle:     "KQkq",
		ep:         -1,
		fullmove:   1,
		status:     "playing",
		message:    "Ván mới — bạn đi Trắng. (New game — you are White.)",
		difficulty: diffMedium,
	}
	start := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	g.setFENBoard(start)
	return g
}

func (g *chessGame) setFENBoard(fenBoard string) {
	for i := range g.board {
		g.board[i] = empty
	}
	rank, file := 7, 0
	for _, ch := range fenBoard {
		switch {
		case ch == '/':
			rank--
			file = 0
		case ch >= '1' && ch <= '8':
			file += int(ch - '0')
		default:
			g.board[rank*8+file] = chessPiece(ch)
			file++
		}
	}
}

func sqName(i int) string {
	return string(rune('a'+i%8)) + string(rune('1'+i/8))
}

func parseSq(s string) (int, bool) {
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

func (g *chessGame) fen() string {
	var b strings.Builder
	for rank := 7; rank >= 0; rank-- {
		emptyN := 0
		for file := 0; file < 8; file++ {
			p := g.board[rank*8+file]
			if p == empty {
				emptyN++
				continue
			}
			if emptyN > 0 {
				b.WriteByte(byte('0' + emptyN))
				emptyN = 0
			}
			b.WriteByte(byte(p))
		}
		if emptyN > 0 {
			b.WriteByte(byte('0' + emptyN))
		}
		if rank > 0 {
			b.WriteByte('/')
		}
	}
	stm := "b"
	if g.white {
		stm = "w"
	}
	cast := g.castle
	if cast == "" {
		cast = "-"
	}
	ep := "-"
	if g.ep >= 0 {
		ep = sqName(g.ep)
	}
	return fmt.Sprintf("%s %s %s %s %d %d", b.String(), stm, cast, ep, g.halfmove, g.fullmove)
}

func (g *chessGame) boardRows() []string {
	rows := make([]string, 8)
	for rank := 7; rank >= 0; rank-- {
		var s strings.Builder
		for file := 0; file < 8; file++ {
			p := g.board[rank*8+file]
			if p == empty {
				s.WriteByte('.')
			} else {
				s.WriteByte(byte(p))
			}
		}
		rows[7-rank] = s.String()
	}
	return rows
}

func (g *chessGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	turn := "black"
	if g.white {
		turn = "white"
	}
	hist := append([]string{}, g.history...)
	return map[string]interface{}{
		"fen":         g.fen(),
		"board":       g.boardRows(),
		"turn":        turn,
		"status":      g.status,
		"winner":      g.winner,
		"lastMove":    g.lastUCI,
		"lastSAN":     g.lastSAN,
		"history":     hist,
		"message":     g.message,
		"youAre":      "white",
		"botIs":       "black",
		"difficulty":  normalizeGameDifficulty(g.difficulty),
		"botThinking": g.botThinking,
	}
}

func (g *chessGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.summaryTextUnlocked()
}

func (g *chessGame) summaryTextUnlocked() string {
	turn := "đen (black / bot)"
	if g.white {
		turn = "trắng (white / người chơi)"
	}
	var b strings.Builder
	b.WriteString("Cờ vua trên web Vector. Người chơi cầm Trắng; bot cầm Đen. ")
	b.WriteString("Lượt hiện tại: " + turn + ". Trạng thái: " + g.status + ". ")
	if g.winner != "" {
		b.WriteString("Người thắng: " + g.winner + ". ")
	}
	if g.lastUCI != "" {
		b.WriteString("Nước gần nhất: " + g.lastUCI)
		if g.lastSAN != "" {
			b.WriteString(" (" + g.lastSAN + ")")
		}
		b.WriteString(". ")
	}
	b.WriteString("FEN: " + g.fen() + ". ")
	b.WriteString("Bàn (hàng 8→1, A→H): ")
	for i, row := range g.boardRows() {
		b.WriteString(fmt.Sprintf("r%d=%s ", 8-i, row))
	}
	if len(g.history) > 0 {
		b.WriteString("Lịch sử UCI: " + strings.Join(g.history, " ") + ".")
	}
	return b.String()
}

func (g *chessGame) reset() map[string]interface{} {
	prev := normalizeGameDifficulty(g.difficulty)
	ng := newChessGame()
	ng.difficulty = prev
	chessMu.Lock()
	chessInst = ng
	chessMu.Unlock()
	return ng.snapshot()
}

func (g *chessGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *chessGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *chessGame) sidePiece() chessPiece {
	if g.white {
		return 'K'
	}
	return 'k'
}

func onBoard(r, f int) bool { return r >= 0 && r < 8 && f >= 0 && f < 8 }

func (g *chessGame) kingSq(white bool) int {
	target := chessPiece('k')
	if white {
		target = 'K'
	}
	for i, p := range g.board {
		if p == target {
			return i
		}
	}
	return -1
}

func (g *chessGame) attacked(sq int, byWhite bool) bool {
	tr, tf := sq/8, sq%8
	// pawns
	if byWhite {
		for _, df := range []int{-1, 1} {
			r, f := tr-1, tf+df
			if onBoard(r, f) && g.board[r*8+f] == 'P' {
				return true
			}
		}
	} else {
		for _, df := range []int{-1, 1} {
			r, f := tr+1, tf+df
			if onBoard(r, f) && g.board[r*8+f] == 'p' {
				return true
			}
		}
	}
	// knights
	knights := [][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}
	wantN := chessPiece('n')
	if byWhite {
		wantN = 'N'
	}
	for _, d := range knights {
		r, f := tr+d[0], tf+d[1]
		if onBoard(r, f) && g.board[r*8+f] == wantN {
			return true
		}
	}
	// king
	wantK := chessPiece('k')
	if byWhite {
		wantK = 'K'
	}
	for dr := -1; dr <= 1; dr++ {
		for df := -1; df <= 1; df++ {
			if dr == 0 && df == 0 {
				continue
			}
			r, f := tr+dr, tf+df
			if onBoard(r, f) && g.board[r*8+f] == wantK {
				return true
			}
		}
	}
	// sliding
	slide := func(dirs [][2]int, bishops, rooks bool) bool {
		for _, d := range dirs {
			r, f := tr+d[0], tf+d[1]
			for onBoard(r, f) {
				p := g.board[r*8+f]
				if p != empty {
					pl := toLower(p)
					okColor := (byWhite && isWhite(p)) || (!byWhite && isBlack(p))
					if okColor {
						if bishops && (pl == 'b' || pl == 'q') {
							return true
						}
						if rooks && (pl == 'r' || pl == 'q') {
							return true
						}
					}
					break
				}
				r += d[0]
				f += d[1]
			}
		}
		return false
	}
	if slide([][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}, true, false) {
		return true
	}
	if slide([][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}, false, true) {
		return true
	}
	return false
}

func (g *chessGame) inCheck(white bool) bool {
	ks := g.kingSq(white)
	if ks < 0 {
		return true
	}
	return g.attacked(ks, !white)
}

func (g *chessGame) genPseudo(from int) []chessMove {
	p := g.board[from]
	if p == empty {
		return nil
	}
	white := isWhite(p)
	if white != g.white {
		return nil
	}
	side := p
	var moves []chessMove
	add := func(to int, promo chessPiece, castle, ep bool) {
		if to < 0 || to > 63 {
			return
		}
		cap := g.board[to]
		if isFriend(cap, side) {
			return
		}
		moves = append(moves, chessMove{
			From: from, To: to, Promo: promo, Capture: cap,
			IsCastle: castle, IsEnPassant: ep,
		})
	}
	r, f := from/8, from%8
	kind := toLower(p)

	switch kind {
	case 'p':
		dir := 1
		startRank := 1
		promoRank := 7
		if !white {
			dir = -1
			startRank = 6
			promoRank = 0
		}
		// forward
		one := (r+dir)*8 + f
		if onBoard(r+dir, f) && g.board[one] == empty {
			if r+dir == promoRank {
				for _, pr := range []chessPiece{'q', 'r', 'b', 'n'} {
					pp := pr
					if white {
						pp = pr - 32
					}
					add(one, pp, false, false)
				}
			} else {
				add(one, 0, false, false)
				if r == startRank {
					two := (r+2*dir)*8 + f
					if g.board[two] == empty {
						add(two, 0, false, false)
					}
				}
			}
		}
		for _, df := range []int{-1, 1} {
			nr, nf := r+dir, f+df
			if !onBoard(nr, nf) {
				continue
			}
			to := nr*8 + nf
			if isEnemy(g.board[to], side) {
				if nr == promoRank {
					for _, pr := range []chessPiece{'q', 'r', 'b', 'n'} {
						pp := pr
						if white {
							pp = pr - 32
						}
						add(to, pp, false, false)
					}
				} else {
					add(to, 0, false, false)
				}
			} else if to == g.ep {
				add(to, 0, false, true)
			}
		}
	case 'n':
		for _, d := range [][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
			nr, nf := r+d[0], f+d[1]
			if onBoard(nr, nf) {
				add(nr*8+nf, 0, false, false)
			}
		}
	case 'b', 'r', 'q':
		var dirs [][2]int
		if kind == 'b' || kind == 'q' {
			dirs = append(dirs, [2]int{1, 1}, [2]int{1, -1}, [2]int{-1, 1}, [2]int{-1, -1})
		}
		if kind == 'r' || kind == 'q' {
			dirs = append(dirs, [2]int{1, 0}, [2]int{-1, 0}, [2]int{0, 1}, [2]int{0, -1})
		}
		for _, d := range dirs {
			nr, nf := r+d[0], f+d[1]
			for onBoard(nr, nf) {
				to := nr*8 + nf
				if isFriend(g.board[to], side) {
					break
				}
				add(to, 0, false, false)
				if g.board[to] != empty {
					break
				}
				nr += d[0]
				nf += d[1]
			}
		}
	case 'k':
		for dr := -1; dr <= 1; dr++ {
			for df := -1; df <= 1; df++ {
				if dr == 0 && df == 0 {
					continue
				}
				nr, nf := r+dr, f+df
				if onBoard(nr, nf) {
					add(nr*8+nf, 0, false, false)
				}
			}
		}
		// castling
		if white && r == 0 && f == 4 {
			if strings.Contains(g.castle, "K") && g.board[5] == empty && g.board[6] == empty &&
				g.board[7] == 'R' && !g.inCheck(true) && !g.attacked(5, false) && !g.attacked(6, false) {
				add(6, 0, true, false)
			}
			if strings.Contains(g.castle, "Q") && g.board[3] == empty && g.board[2] == empty && g.board[1] == empty &&
				g.board[0] == 'R' && !g.inCheck(true) && !g.attacked(3, false) && !g.attacked(2, false) {
				add(2, 0, true, false)
			}
		}
		if !white && r == 7 && f == 4 {
			if strings.Contains(g.castle, "k") && g.board[61] == empty && g.board[62] == empty &&
				g.board[63] == 'r' && !g.inCheck(false) && !g.attacked(61, true) && !g.attacked(62, true) {
				add(62, 0, true, false)
			}
			if strings.Contains(g.castle, "q") && g.board[59] == empty && g.board[58] == empty && g.board[57] == empty &&
				g.board[56] == 'r' && !g.inCheck(false) && !g.attacked(59, true) && !g.attacked(58, true) {
				add(58, 0, true, false)
			}
		}
	}
	return moves
}

func (g *chessGame) apply(m chessMove) {
	m.PrevEP = g.ep
	m.PrevCastle = g.castle
	m.PrevHalfmove = g.halfmove
	p := g.board[m.From]
	g.board[m.From] = empty
	if m.IsEnPassant {
		if g.white {
			g.board[m.To-8] = empty
		} else {
			g.board[m.To+8] = empty
		}
	}
	if m.IsCastle {
		switch m.To {
		case 6: // white O-O
			g.board[7] = empty
			g.board[5] = 'R'
		case 2:
			g.board[0] = empty
			g.board[3] = 'R'
		case 62:
			g.board[63] = empty
			g.board[61] = 'r'
		case 58:
			g.board[56] = empty
			g.board[59] = 'r'
		}
	}
	if m.Promo != 0 {
		g.board[m.To] = m.Promo
	} else {
		g.board[m.To] = p
	}
	// ep target
	g.ep = -1
	if toLower(p) == 'p' && abs(m.To/8-m.From/8) == 2 {
		g.ep = (m.From+m.To)/2
	}
	// castling rights
	strip := func(ch string) {
		g.castle = strings.ReplaceAll(g.castle, ch, "")
	}
	if p == 'K' {
		strip("K")
		strip("Q")
	}
	if p == 'k' {
		strip("k")
		strip("q")
	}
	if m.From == 0 || m.To == 0 {
		strip("Q")
	}
	if m.From == 7 || m.To == 7 {
		strip("K")
	}
	if m.From == 56 || m.To == 56 {
		strip("q")
	}
	if m.From == 63 || m.To == 63 {
		strip("k")
	}
	if toLower(p) == 'p' || m.Capture != empty || m.IsEnPassant {
		g.halfmove = 0
	} else {
		g.halfmove++
	}
	if !g.white {
		g.fullmove++
	}
	g.white = !g.white
}

func (g *chessGame) undo(m chessMove) {
	g.white = !g.white
	if !g.white {
		g.fullmove--
	}
	g.ep = m.PrevEP
	g.castle = m.PrevCastle
	g.halfmove = m.PrevHalfmove
	p := g.board[m.To]
	if m.Promo != 0 {
		if g.white {
			p = 'P'
		} else {
			p = 'p'
		}
	}
	g.board[m.To] = empty
	g.board[m.From] = p
	if m.IsEnPassant {
		if g.white {
			g.board[m.To-8] = 'p'
		} else {
			g.board[m.To+8] = 'P'
		}
	} else if m.Capture != empty {
		g.board[m.To] = m.Capture
	}
	if m.IsCastle {
		switch m.To {
		case 6:
			g.board[5] = empty
			g.board[7] = 'R'
		case 2:
			g.board[3] = empty
			g.board[0] = 'R'
		case 62:
			g.board[61] = empty
			g.board[63] = 'r'
		case 58:
			g.board[59] = empty
			g.board[56] = 'r'
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (g *chessGame) legalMoves() []chessMove {
	var out []chessMove
	white := g.white
	for from := 0; from < 64; from++ {
		for _, m := range g.genPseudo(from) {
			g.apply(m)
			ok := !g.inCheck(white)
			g.undo(m)
			if ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func (g *chessGame) refreshStatus() {
	legal := g.legalMoves()
	check := g.inCheck(g.white)
	if len(legal) == 0 {
		if check {
			g.status = "checkmate"
			if g.white {
				g.winner = "black"
				g.message = "Chiếu hết — Đen (bot) thắng. (Checkmate — Black wins.)"
			} else {
				g.winner = "white"
				g.message = "Chiếu hết — Trắng thắng. (Checkmate — White wins.)"
			}
		} else {
			g.status = "stalemate"
			g.winner = ""
			g.message = "Hết nước — hòa. (Stalemate — draw.)"
		}
		return
	}
	if check {
		g.status = "check"
		g.message = "Đang chiếu! (Check!)"
	} else {
		g.status = "playing"
		if g.white {
			g.message = "Lượt Trắng (bạn). (White to move.)"
		} else {
			g.message = "Lượt Đen (bot). (Black to move.)"
		}
	}
}

func moveUCI(m chessMove) string {
	u := sqName(m.From) + sqName(m.To)
	if m.Promo != 0 {
		u += string(toLower(m.Promo))
	}
	return u
}

func (g *chessGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.botThinking {
		g.mu.Unlock()
		return nil, fmt.Errorf("bot thinking")
	}
	if g.status == "checkmate" || g.status == "stalemate" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	// Human always plays white.
	if !g.white {
		g.mu.Unlock()
		return nil, fmt.Errorf("not your turn")
	}
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad move")
	}
	from, ok1 := parseSq(uci[0:2])
	to, ok2 := parseSq(uci[2:4])
	if !ok1 || !ok2 {
		g.mu.Unlock()
		return nil, fmt.Errorf("bad squares")
	}
	var promo chessPiece
	if len(uci) >= 5 {
		promo = chessPiece(uci[4])
		if g.white {
			promo = promo - 32
		}
	}
	var chosen *chessMove
	for _, m := range g.legalMoves() {
		if m.From == from && m.To == to {
			if promo == 0 || m.Promo == promo || (promo != 0 && toLower(m.Promo) == toLower(promo)) {
				mm := m
				if promo != 0 {
					mm.Promo = promo
				} else if m.Promo != 0 {
					// default queen
					if g.white {
						mm.Promo = 'Q'
					} else {
						mm.Promo = 'q'
					}
				}
				chosen = &mm
				break
			}
		}
	}
	if chosen == nil {
		g.mu.Unlock()
		return nil, fmt.Errorf("illegal move")
	}
	youPiece := g.board[chosen.From]
	g.apply(*chosen)
	u := moveUCI(*chosen)
	g.lastUCI = u
	g.lastSAN = u
	g.history = append(g.history, u)
	youMove := u
	g.refreshStatus()
	needBot := g.status != "checkmate" && g.status != "stalemate" && !g.white
	var thinkGen int
	statusAfterYou := g.status
	winnerAfterYou := g.winner
	if needBot {
		g.botThinking = true
		g.thinkGen++
		thinkGen = g.thinkGen
		g.message = "Bot đang suy nghĩ…"
	}

	turn := "black"
	if g.white {
		turn = "white"
	}
	hist := append([]string{}, g.history...)
	resp := map[string]interface{}{
		"fen": g.fen(), "board": g.boardRows(), "turn": turn,
		"status": g.status, "winner": g.winner, "lastMove": g.lastUCI,
		"lastSAN": g.lastSAN, "history": hist, "message": g.message,
		"youAre": "white", "botIs": "black",
		"youMove": youMove, "botMove": "",
		"youPiece": string(youPiece), "botPiece": "",
		"difficulty":  normalizeGameDifficulty(g.difficulty),
		"botThinking": needBot,
	}
	g.mu.Unlock()

	// Read human move right away (don't wait for bot).
	if youMove != "" {
		// While bot will reply: speak only your move (and check if any). Game-over full line if no bot.
		stSpeak := statusAfterYou
		winSpeak := winnerAfterYou
		if needBot && stSpeak != "checkmate" && stSpeak != "stalemate" {
			// Don't announce checkmate here (bot still to move); still announce "Check!" for black in check.
			if stSpeak != "check" {
				stSpeak = "playing"
			}
			winSpeak = ""
		}
		// Hint so Xiaozhi routes to self.chess.* (not self.xiangqi.*).
		chessHint := "Cờ vua. Nước người chơi: " + youMove + "."
		queueGameSpeak("chess", buildChessSpokenComment(youMove, "", string(youPiece), "", stSpeak, winSpeak), chessHint)
	}

	// Bot thinks async (~2s) then moves + speaks its own move.
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

// runBotThink waits ~2s then plays Black and announces the bot move only.
func (g *chessGame) runBotThink(gen int) {
	time.Sleep(2 * time.Second)
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	botMove := ""
	var botPiece chessPiece
	if g.status != "checkmate" && g.status != "stalemate" && !g.white {
		botPiece = g.botMoveLocked()
		botMove = g.lastUCI
	}
	g.botThinking = false
	// Prefer a clear "your turn" label when play continues.
	if g.white && (g.status == "playing" || g.status == "check") {
		if g.status == "check" {
			g.message = "Đến lượt bạn — đang chiếu!"
		} else {
			g.message = "Đến lượt bạn."
		}
	}
	status := g.status
	winner := g.winner
	summary := g.summaryTextUnlocked()
	g.mu.Unlock()

	if botMove != "" {
		queueGameSpeak("chess", buildChessSpokenComment("", botMove, "", string(botPiece), status, winner), summary)
	}
}

func (g *chessGame) botMoveLocked() chessPiece {
	legal := g.legalMoves()
	if len(legal) == 0 {
		g.refreshStatus()
		return 0
	}
	var best chessMove
	switch normalizeGameDifficulty(g.difficulty) {
	case diffEasy:
		// Was roughly "old hard" (still shallow) + noise — intentionally weak.
		best = g.pickChessSearch(legal, 1, 28)
	case diffHard:
		// Strong: iterative search + SEE filter (no hung queen gift).
		best = g.pickChessHard(legal)
	default:
		// Medium: solid 2–3 ply, tiny noise, still skip SEE≪0 captures.
		best = g.pickChessSearch(legal, 3, 6)
	}
	piece := g.board[best.From]
	g.apply(best)
	u := moveUCI(best)
	g.lastUCI = u
	g.lastSAN = u
	g.history = append(g.history, u)
	g.refreshStatus()
	return piece
}

// staticEvalWhitePOV: positive = White is better (centipawn-ish).
func (g *chessGame) staticEvalWhitePOV() int {
	score := 0
	wMat, bMat := 0, 0
	for i, p := range g.board {
		if p == empty {
			continue
		}
		v := materialCP(p) + chessPST(p, i)
		if isWhite(p) {
			score += v
			wMat += materialCP(p)
		} else {
			score -= v
			bMat += materialCP(p)
		}
	}
	// Mild mobility proxy via attacked empty/center — skip heavy gen.
	// Discourage naked king walk / back-rank only via PST above.
	_ = wMat
	_ = bMat
	return score
}

func chessPST(p chessPiece, sq int) int {
	// Mild center / advance bonuses. Rank 0 = white back, 7 = black back.
	r, f := sq/8, sq%8
	center := 0
	if f >= 2 && f <= 5 && r >= 2 && r <= 5 {
		center = 6
	}
	switch toLower(p) {
	case 'p':
		if isWhite(p) {
			return r*4 + center/2
		}
		return (7-r)*4 + center/2
	case 'n', 'b':
		return center + 4
	case 'q':
		// Keep queen nearer home early-ish; discourage mid-board "dives".
		if isWhite(p) {
			if r <= 1 {
				return 2
			}
			if r >= 5 {
				return -8
			}
			return center / 2
		}
		if r >= 6 {
			return 2
		}
		if r <= 2 {
			return -8
		}
		return center / 2
	case 'r':
		return center / 2
	case 'k':
		// Prefer safer king early (home rank + castled files).
		if r == 0 || r == 7 {
			if f == 1 || f == 2 || f == 6 {
				return 10
			}
			return 6
		}
		return -12
	default:
		return center / 2
	}
}

// evaluateSTM: score for the side about to move.
func (g *chessGame) evaluateSTM() int {
	s := g.staticEvalWhitePOV()
	// Small hang penalty: our piece attacked on opponent's turn eye.
	// (side to move can often take free stuff first — so only soft weight.)
	sHang := g.hangSoftSTM()
	if g.white {
		return s + sHang
	}
	return -s + sHang
}

// hangSoftSTM: negative if side-to-move leaves material en prise after "standing".
// Approximate: for each of our pieces attacked by opponent not defended by us.
func (g *chessGame) hangSoftSTM() int {
	pen := 0
	for i, p := range g.board {
		if p == empty {
			continue
		}
		ours := (g.white && isWhite(p)) || (!g.white && isBlack(p))
		if !ours {
			continue
		}
		if !g.attacked(i, !g.white) {
			continue
		}
		// Attacked by opponent.
		defended := g.attacked(i, g.white)
		v := materialCP(p)
		if v <= 0 {
			continue // king
		}
		if !defended {
			pen -= v / 2 // soft: half value if unprotected hang
		} else if v >= 900 {
			// Queen under fire even if defended — still a little careless.
			pen -= 30
		}
	}
	return pen
}

func (g *chessGame) pickChessEasy(legal []chessMove) chessMove {
	// Often random; sometimes grab a free-ish capture.
	if chessRNG.Intn(100) < 55 {
		return legal[chessRNG.Intn(len(legal))]
	}
	best := legal[chessRNG.Intn(len(legal))]
	bestScore := -99999
	for _, m := range legal {
		score := 0
		if m.Capture != empty || m.IsEnPassant {
			score += 10 + material(m.Capture)
		}
		// 40% chance to ignore this capture (blunder)
		if score > 0 && chessRNG.Intn(100) < 40 {
			score = 0
		}
		if score > bestScore {
			bestScore = score
			best = m
		}
	}
	if bestScore <= 0 {
		return legal[chessRNG.Intn(len(legal))]
	}
	return best
}

// chessSEE: one-ply static exchange — positive = we keep the material, negative = hung gift.
// Example: QxQ then opponent recaptures our Q → 0 (trade) or negative if over-invested.
func (g *chessGame) chessSEE(m chessMove) int {
	mover := g.board[m.From]
	if mover == empty {
		return 0
	}
	gain := materialCP(m.Capture)
	if m.IsEnPassant {
		gain = materialCP('p')
	}
	if m.Promo != 0 {
		// net of pawn→promo: promo value minus pawn (already on board)
		gain += materialCP(m.Promo) - materialCP('p')
	}
	g.apply(m)
	// After move, side to move is the opponent.
	toPiece := g.board[m.To]
	if toPiece != empty && g.attacked(m.To, g.white) {
		// Opponent can take us back (approx full MVV — ignore recapture chains).
		gain -= materialCP(toPiece)
	}
	// Giving check allows a mild negative see (tempo), but not dumping a full queen.
	checking := g.inCheck(g.white)
	g.undo(m)
	if checking && gain >= -materialCP(mover)/2 {
		if gain < 0 {
			gain += 40
		}
	}
	// Pure equal queen trades are never "good" — slight penalty so search
	// prefers developing when scores are otherwise equal.
	if gain == 0 && materialCP(m.Capture) >= 900 {
		gain = -25
	}
	return gain
}

func chessMoveOrder(m chessMove) int {
	s := 0
	if m.Capture != empty || m.IsEnPassant {
		s += 100 + material(m.Capture)*10
	}
	if m.Promo != 0 {
		s += 80
	}
	return s
}

func (g *chessGame) orderMovesChess(legal []chessMove) []chessMove {
	order := make([]chessMove, len(legal))
	copy(order, legal)
	type scored struct {
		m chessMove
		s int
	}
	ss := make([]scored, len(order))
	for i, m := range order {
		s := chessMoveOrder(m)
		if m.Capture != empty || m.IsEnPassant {
			s += g.chessSEE(m) / 10
		}
		ss[i] = scored{m, s}
	}
	for i := 0; i < len(ss); i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j].s > ss[i].s {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
	for i := range order {
		order[i] = ss[i].m
	}
	return order
}

func (g *chessGame) pickChessSearch(legal []chessMove, depth int, noisePct int) chessMove {
	if depth < 1 {
		depth = 1
	}
	order := g.orderMovesChess(legal)
	// Medium: drop captures that clearly hang more than a pawn.
	if noisePct < 20 {
		filt := make([]chessMove, 0, len(order))
		for _, m := range order {
			if (m.Capture != empty || m.IsEnPassant) && g.chessSEE(m) < -50 {
				continue
			}
			filt = append(filt, m)
		}
		if len(filt) > 0 {
			order = filt
		}
	}
	best := order[0]
	bestScore := -999999
	bestSEE := -999999
	alpha, beta := -999999, 999999
	for _, m := range order {
		g.apply(m)
		sc := -g.negamaxChess(depth-1, -beta, -alpha, true)
		g.undo(m)
		see := 0
		if m.Capture != empty || m.IsEnPassant {
			see = g.chessSEE(m)
		}
		// Bias against hung / dead-equal trades when search ties.
		scAdj := sc + see/4
		if scAdj > bestScore || (scAdj == bestScore && see > bestSEE) {
			bestScore = scAdj
			bestSEE = see
			best = m
		}
		if sc > alpha {
			alpha = sc
		}
	}
	// Noise: occasionally pick a slightly worse move (easy/medium only).
	if noisePct > 0 && chessRNG.Intn(100) < noisePct && len(order) > 1 {
		pool := make([]chessMove, 0, 6)
		for _, m := range order {
			if (m.Capture != empty || m.IsEnPassant) && g.chessSEE(m) < -100 {
				continue
			}
			g.apply(m)
			sc := -g.negamaxChess(depth-1, -999999, 999999, true)
			g.undo(m)
			if sc >= bestScore-80 {
				pool = append(pool, m)
			}
		}
		if len(pool) > 0 {
			return pool[chessRNG.Intn(len(pool))]
		}
	}
	return best
}

// pickChessHard: iterative deepening + SEE root filter — no random blunders, no hung QxQ.
func (g *chessGame) pickChessHard(legal []chessMove) chessMove {
	order := g.orderMovesChess(legal)
	filtered := make([]chessMove, 0, len(order))
	for _, m := range order {
		if (m.Capture != empty || m.IsEnPassant) && g.chessSEE(m) < 0 {
			// Skip hanging gifts (QxQ into recapture counts as <=0 after equal-trade pen).
			continue
		}
		filtered = append(filtered, m)
	}
	if len(filtered) == 0 {
		filtered = order
	} else {
		order = filtered
	}
	best := order[0]
	const maxDepth = 4
	for depth := 1; depth <= maxDepth; depth++ {
		bestScore := -999999
		bestSEE := -999999
		alpha, beta := -999999, 999999
		localBest := best
		for _, m := range order {
			see := 0
			if m.Capture != empty || m.IsEnPassant {
				see = g.chessSEE(m)
				// At every root iter, still refuse deep-negative SEE.
				if see < -30 {
					continue
				}
			}
			g.apply(m)
			ext := 0
			// Only extend checks that aren't pure material hangs.
			if g.inCheck(g.white) && see >= 0 {
				ext = 1
			}
			sc := -g.negamaxChess(depth-1+ext, -beta, -alpha, true)
			if sc > 40000 {
				sc += depth
			}
			// Soft root SEE bias so free captures beat dead-equal Q trades.
			sc += see / 5
			g.undo(m)
			if sc > bestScore || (sc == bestScore && see > bestSEE) {
				bestScore = sc
				bestSEE = see
				localBest = m
			}
			if sc > alpha {
				alpha = sc
			}
		}
		best = localBest
		// Move best to front for next iteration.
		for i, m := range order {
			if m.From == best.From && m.To == best.To && m.Promo == best.Promo {
				order[0], order[i] = order[i], order[0]
				break
			}
		}
	}
	return best
}

func (g *chessGame) negamaxChess(depth, alpha, beta int, doQ bool) int {
	if depth <= 0 {
		if doQ {
			return g.quiesceChess(alpha, beta, 4)
		}
		return g.evaluateSTM()
	}
	legal := g.legalMoves()
	if len(legal) == 0 {
		if g.inCheck(g.white) {
			return -50000 + depth // prefer faster mates later
		}
		return 0
	}
	legal = g.orderMovesChess(legal)
	best := -999999
	for _, m := range legal {
		// Prune obviously losing captures mid-tree (still allow quiet).
		if depth >= 2 && (m.Capture != empty || m.IsEnPassant) && g.chessSEE(m) < -200 {
			continue
		}
		g.apply(m)
		ext := 0
		if depth >= 2 && g.inCheck(g.white) {
			// Don't extend if we just hung a major (cheap check).
			// (see after apply: piece on m.To)
			hungMajor := false
			if p := g.board[m.To]; p != empty && materialCP(p) >= 500 && g.attacked(m.To, g.white) {
				// opponent to move and attacks our just-moved piece
				hungMajor = true
			}
			if !hungMajor {
				ext = 1
			}
		}
		sc := -g.negamaxChess(depth-1+ext, -beta, -alpha, doQ)
		g.undo(m)
		if sc > best {
			best = sc
		}
		if sc > alpha {
			alpha = sc
		}
		if alpha >= beta {
			break
		}
	}
	if best <= -999990 {
		// Everything pruned — search one quiet-ish move raw.
		for _, m := range legal {
			g.apply(m)
			sc := -g.negamaxChess(depth-1, -beta, -alpha, doQ)
			g.undo(m)
			if sc > best {
				best = sc
			}
			if sc > alpha {
				alpha = sc
			}
			if alpha >= beta {
				break
			}
		}
	}
	return best
}

// quiesceChess: capture-only search; skip SEE-negative captures (delta pruning).
func (g *chessGame) quiesceChess(alpha, beta, qDepth int) int {
	stand := g.evaluateSTM()
	if stand >= beta {
		return beta
	}
	if stand > alpha {
		alpha = stand
	}
	if qDepth <= 0 {
		return stand
	}
	legal := g.legalMoves()
	// MVV order
	for i := 0; i < len(legal); i++ {
		for j := i + 1; j < len(legal); j++ {
			if chessMoveOrder(legal[j]) > chessMoveOrder(legal[i]) {
				legal[i], legal[j] = legal[j], legal[i]
			}
		}
	}
	for _, m := range legal {
		if m.Capture == empty && !m.IsEnPassant {
			continue
		}
		see := g.chessSEE(m)
		if see < 0 {
			continue // don't chase hung recaptures / dead trades that lose
		}
		// Delta: even best capture can't raise stand to alpha.
		if stand+see+50 < alpha {
			continue
		}
		g.apply(m)
		sc := -g.quiesceChess(-beta, -alpha, qDepth-1)
		g.undo(m)
		if sc >= beta {
			return beta
		}
		if sc > alpha {
			alpha = sc
		}
	}
	return alpha
}

// material returns small integer values for ordering.
func material(p chessPiece) int {
	switch toLower(p) {
	case 'p':
		return 1
	case 'n', 'b':
		return 3
	case 'r':
		return 5
	case 'q':
		return 9
	case 'k':
		return 0
	default:
		return 0
	}
}

// materialCP is centipawn-ish material for eval / SEE.
func materialCP(p chessPiece) int {
	switch toLower(p) {
	case 'p':
		return 100
	case 'n', 'b':
		return 320
	case 'r':
		return 500
	case 'q':
		return 900
	case 'k':
		return 0
	default:
		return 0
	}
}

func (g *chessGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.botThinking || !g.white {
		return []string{}
	}
	ms := g.legalMoves()
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, moveUCI(m))
	}
	return out
}
