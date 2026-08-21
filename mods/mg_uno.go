package mods

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Simplified 2-player Uno. Cards are "<color><rank>": colors R,G,B,Y with
// ranks 0-9, S(skip), D(draw2), V(reverse); wild cards are "W" and "W4"
// (no color until chosen). 7 cards dealt to each side.

type unoGame struct {
	miniCommon
	deck          []string
	discard       []string
	hand          []string // human
	botHand       []string
	top           string
	color         string // active color (also drives non-wild top's color)
	direction     int    // +1 clockwise, -1 counter (flip on reverse)
	pendingColor  bool
	pendingDraw4  bool
	humanLastCard string
	botLastCard   string
	botTurnCards  []string // cards robot played in the last think combo
	botForcedDraw int      // cards human was forced to draw in that combo
	botForceCard  string   // the +2 / +4 that caused the forced draw
	drawAnimTo    string   // "human" | "bot" | "" — who receives forced draws
	drawAnimN     int      // how many cards to animate drawing
	botMustCont   bool     // bot still has another play after draw/skip pause
	saidUno       bool     // human called UNO this hand
	justDrew      bool     // human drew and may pass or play
}

func unoPick(vi, en []string) (string, string) {
	if len(vi) == 0 {
		return "", ""
	}
	i := rand.Intn(len(vi))
	j := i
	if len(en) > 0 {
		j = rand.Intn(len(en))
	}
	e := ""
	if len(en) > 0 {
		e = en[j]
	}
	return vi[i], e
}

var (
	unoMu   sync.Mutex
	unoInst *unoGame
)

func getUno() *unoGame {
	unoMu.Lock()
	defer unoMu.Unlock()
	if unoInst == nil {
		unoInst = newUnoGame()
	}
	return unoInst
}

func unoColorVI(col string) string {
	switch strings.ToUpper(col) {
	case "R":
		return "đỏ"
	case "G":
		return "xanh lá"
	case "B":
		return "xanh dương"
	case "Y":
		return "vàng"
	default:
		return col
	}
}

func unoColorEN(col string) string {
	switch strings.ToUpper(col) {
	case "R":
		return "red"
	case "G":
		return "green"
	case "B":
		return "blue"
	case "Y":
		return "yellow"
	default:
		return col
	}
}

// unoSpeakCard returns a natural spoken description of a card code.
// For wilds, pass chosenColor (R/G/B/Y) after the player/bot picks a color.
func unoSpeakCard(card, chosenColor string) (vi, en string) {
	c := strings.ToUpper(strings.TrimSpace(card))
	if c == "W" {
		vi, en = "đổi màu", "wild"
		if chosenColor != "" {
			vi += ", chọn " + unoColorVI(chosenColor)
			en += ", chose " + unoColorEN(chosenColor)
		}
		return
	}
	if c == "W4" {
		vi, en = "cộng bốn", "draw four"
		if chosenColor != "" {
			vi += ", chọn " + unoColorVI(chosenColor)
			en += ", chose " + unoColorEN(chosenColor)
		}
		return
	}
	if len(c) < 2 {
		return c, c
	}
	col, rank := c[:1], c[1:]
	cv, ce := unoColorVI(col), unoColorEN(col)
	switch rank {
	case "S":
		return "bỏ lượt " + cv, "skip " + ce
	case "D":
		return "cộng hai " + cv, "draw two " + ce
	case "V":
		return "đảo chiều " + cv, "reverse " + ce
	default:
		return rank + " " + cv, rank + " " + ce
	}
}

func unoSpeakCardMode(card, chosenColor string) string {
	vi, en := unoSpeakCard(card, chosenColor)
	if chessPreferVIText() {
		return vi
	}
	return en
}

func unoNewDeck() []string {
	colors := []string{"R", "G", "B", "Y"}
	var deck []string
	for _, c := range colors {
		deck = append(deck, c+"0")
		for n := 1; n <= 9; n++ {
			deck = append(deck, c+fmt.Sprintf("%d", n))
			deck = append(deck, c+fmt.Sprintf("%d", n))
		}
		for i := 0; i < 2; i++ {
			deck = append(deck, c+"S") // skip
			deck = append(deck, c+"D") // draw 2
			deck = append(deck, c+"V") // reverse (2-player ≈ skip)
		}
	}
	for i := 0; i < 4; i++ {
		deck = append(deck, "W")
		deck = append(deck, "W4")
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

// unoDrawFirstNumberCard removes and returns the first plain number card
// found in deck, so the opening discard is never an action/wild card.
func unoDrawFirstNumberCard(deck *[]string) string {
	for i, c := range *deck {
		if !strings.HasPrefix(c, "W") && len(c) == 2 && c[1] >= '0' && c[1] <= '9' {
			*deck = append((*deck)[:i], (*deck)[i+1:]...)
			return c
		}
	}
	if len(*deck) > 0 {
		c := (*deck)[0]
		*deck = (*deck)[1:]
		return c
	}
	return "R0"
}

func newUnoGame() *unoGame {
	deck := unoNewDeck()
	hand := append([]string{}, deck[:7]...)
	deck = deck[7:]
	bot := append([]string{}, deck[:7]...)
	deck = deck[7:]
	top := unoDrawFirstNumberCard(&deck)

	g := &unoGame{
		deck:      deck,
		hand:      hand,
		botHand:   bot,
		top:       top,
		color:     top[0:1],
		direction: 1,
		discard:   []string{top},
	}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	vi, en := unoPick(
		[]string{
			"Hãy bắt đầu nào! Bạn có 7 lá.",
			"Tôi sẽ cố gắng thắng. Đến lượt bạn.",
			"Tôi chia bài nhé — chúc bạn may mắn!",
			"Uno mới! Cẩn thận với +4 của tôi.",
			"Bắt đầu thôi. Đánh theo màu hoặc số.",
			"Ván mới — hy vọng hôm nay bạn may mắn.",
			"Tôi sẵn sàng. Lượt bạn đi trước.",
			"Chia xong 7 lá mỗi bên. Bắt đầu!",
		},
		[]string{
			"Let's go! You have 7 cards.",
			"I'll try to win. Your turn.",
			"I dealt the cards — good luck!",
			"New Uno! Watch out for my +4.",
			"Start by matching color or number.",
			"Fresh hand — hope luck is with you.",
			"I'm ready. You go first.",
			"Seven each. Let's play!",
		},
	)
	g.message, _ = viOrEN(vi, en)
	return g
}

func (g *unoGame) canPlay(card string) bool {
	if strings.HasPrefix(card, "W") {
		return true
	}
	if len(card) < 2 {
		return false
	}
	color := card[0:1]
	rank := card[1:]
	topRank := ""
	if len(g.top) > 1 && !strings.HasPrefix(g.top, "W") {
		topRank = g.top[1:]
	}
	return color == g.color || (topRank != "" && rank == topRank)
}

// drawCardLocked pops one card from the deck, reshuffling the discard pile
// (minus the current top) back into the deck if it runs out.
func (g *unoGame) drawCardLocked() string {
	if len(g.deck) == 0 {
		if len(g.discard) <= 1 {
			return ""
		}
		top := g.discard[len(g.discard)-1]
		reshuffled := append([]string{}, g.discard[:len(g.discard)-1]...)
		rand.Shuffle(len(reshuffled), func(i, j int) { reshuffled[i], reshuffled[j] = reshuffled[j], reshuffled[i] })
		g.deck = reshuffled
		g.discard = []string{top}
	}
	if len(g.deck) == 0 {
		return ""
	}
	c := g.deck[len(g.deck)-1]
	g.deck = g.deck[:len(g.deck)-1]
	return c
}

func (g *unoGame) drawNLocked(target *[]string, n int) {
	for i := 0; i < n; i++ {
		if c := g.drawCardLocked(); c != "" {
			*target = append(*target, c)
		}
	}
}

func (g *unoGame) botPickColorLocked() string {
	counts := map[string]int{"R": 0, "G": 0, "B": 0, "Y": 0}
	for _, c := range g.botHand {
		if !strings.HasPrefix(c, "W") {
			counts[c[0:1]]++
		}
	}
	best, bestN := "R", -1
	for _, col := range []string{"R", "G", "B", "Y"} {
		if counts[col] > bestN {
			best, bestN = col, counts[col]
		}
	}
	return best
}

// botPlayOneStepLocked plays a single bot action. Returns:
//
//	"draw_pause" — played +2/+4; top card is on the pile; wait for draw anim then continue
//	"again"      — skip/reverse; brief pause then continue
//	"done"       — turn handed to human (or game over / bot could not play)
func (g *unoGame) botPlayOneStepLocked() string {
	g.drawAnimTo = ""
	g.drawAnimN = 0

	idx := -1
	for i, c := range g.botHand {
		if g.canPlay(c) {
			idx = i
			break
		}
	}
	if idx == -1 {
		if c := g.drawCardLocked(); c != "" {
			g.botHand = append(g.botHand, c)
			if g.canPlay(c) {
				idx = len(g.botHand) - 1
			}
		}
	}
	if idx == -1 {
		if len(g.botTurnCards) == 0 {
			g.message = unoBotTurnMessage(nil, 0, "", len(g.botHand))
		} else {
			g.message = unoBotTurnMessage(g.botTurnCards, g.botForcedDraw, g.botForceCard, len(g.botHand))
		}
		g.humanTurn = true
		g.botMustCont = false
		return "done"
	}

	card := g.botHand[idx]
	g.botHand = append(g.botHand[:idx], g.botHand[idx+1:]...)
	g.top = card
	g.discard = append(g.discard, card)
	g.lastMove = "bot:" + card
	g.botLastCard = card
	g.botTurnCards = append(g.botTurnCards, card)

	if strings.HasPrefix(card, "W") {
		g.color = g.botPickColorLocked()
		if card == "W4" {
			g.drawNLocked(&g.hand, 4)
			g.botForcedDraw += 4
			g.botForceCard = card
			g.drawAnimTo = "human"
			g.drawAnimN = 4
			g.botMustCont = true
			vi, en := unoPick(
				[]string{"Đây mới là bất ngờ — cộng bốn!", "Bạn phải rút bốn lá.", "Wild +4 — xin lỗi nhé.", "Kế hoạch khá hay… với tôi.", "Rút bốn đi! Tôi chọn màu."},
				[]string{"Surprise — draw four!", "You draw four cards.", "Wild +4 — sorry.", "Nice plan… for me.", "Draw four! Color set."},
			)
			g.message, _ = viOrEN(vi, en)
			if len(g.botHand) == 0 {
				g.status = "win"
				g.winner = "bot"
				g.message = unoWinMessage(false)
				g.humanTurn = false
				g.botMustCont = false
				return "done"
			}
			return "draw_pause"
		}
	} else {
		g.color = card[0:1]
		switch card[1:] {
		case "S", "V":
			if card[1:] == "V" {
				g.direction = -g.direction
				if g.direction == 0 {
					g.direction = 1
				}
			}
			g.botMustCont = true
			vi, en := unoPick(
				[]string{"Lượt của bạn bị bỏ!", "Bạn vừa bị chặn.", "Skip! Tôi đánh tiếp…", "Đổi chiều — tôi giữ lượt.", "Không đến lượt bạn đâu."},
				[]string{"You're skipped!", "Blocked.", "Skip — I continue…", "Reverse — I keep going.", "Not your turn."},
			)
			g.message, _ = viOrEN(vi, en)
			if len(g.botHand) == 0 {
				g.status = "win"
				g.winner = "bot"
				g.message = unoWinMessage(false)
				g.humanTurn = false
				g.botMustCont = false
				return "done"
			}
			return "again"
		case "D":
			g.drawNLocked(&g.hand, 2)
			g.botForcedDraw += 2
			g.botForceCard = card
			g.drawAnimTo = "human"
			g.drawAnimN = 2
			g.botMustCont = true
			vi, en := unoPick(
				[]string{"Nhận lấy hai lá nhé!", "Cộng hai — rút đi.", "Kế hoạch khá hay.", "+2 cho bạn.", "Bạn đang rút hai lá…"},
				[]string{"Take two cards!", "Draw two.", "Nice plan.", "+2 for you.", "Drawing two…"},
			)
			g.message, _ = viOrEN(vi, en)
			if len(g.botHand) == 0 {
				g.status = "win"
				g.winner = "bot"
				g.message = unoWinMessage(false)
				g.humanTurn = false
				g.botMustCont = false
				return "done"
			}
			return "draw_pause"
		}
	}

	if len(g.botHand) == 0 {
		g.status = "win"
		g.winner = "bot"
		g.message = unoWinMessage(false)
		g.humanTurn = false
		g.botMustCont = false
		return "done"
	}

	g.botMustCont = false
	g.humanTurn = true
	g.message = unoBotTurnMessage(g.botTurnCards, g.botForcedDraw, g.botForceCard, len(g.botHand))
	return "done"
}

func unoWinMessage(humanWins bool) string {
	if humanWins {
		vi, en := unoPick(
			[]string{"Xuất sắc! Bạn thắng rồi.", "Tuyệt vời — tôi sẽ phục thù.", "Bạn thắng Uno!", "Chúc mừng chiến thắng!", "Bạn chơi quá hay."},
			[]string{"Brilliant! You win.", "Wow — I'll get revenge.", "You win Uno!", "Congratulations!", "You played great."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	}
	vi, en := unoPick(
		[]string{"Tôi thắng rồi!", "Ván sau nhé.", "Cố lên lần tới.", "Robot hết bài — tôi thắng.", "May mắn chưa về phía bạn."},
		[]string{"I win!", "Next game.", "Better luck next time.", "I'm out — robot wins.", "Luck wasn't with you."},
	)
	msg, _ := viOrEN(vi, en)
	return msg
}

// unoBotTurnMessage explains the whole bot combo (e.g. +2 then another card)
// so the UI/speak never hide why the human suddenly got cards.
func unoBotTurnMessage(played []string, forcedDraw int, forceCard string, botLeft int) string {
	if len(played) == 0 {
		vi, en := unoPick(
			[]string{"Ôi không — tôi phải rút bài.", "Tôi không đánh được.", "Bạn chơi khá đấy, tôi rút.", "Hmm, lượt của bạn.", "May cho bạn, tôi bí bài."},
			[]string{"Oh no — I have to draw.", "I can't play.", "Nice play — I draw.", "Hmm, your turn.", "Lucky you, I'm stuck."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	}
	partsVI := make([]string, 0, len(played))
	partsEN := make([]string, 0, len(played))
	for _, c := range played {
		vi, en := unoSpeakCard(c, "")
		partsVI = append(partsVI, vi)
		partsEN = append(partsEN, en)
	}
	seqVI := strings.Join(partsVI, " → ")
	seqEN := strings.Join(partsEN, " → ")

	// Near-win pressure
	if botLeft == 1 {
		vi, en := unoPick(
			[]string{"Tôi chỉ còn một lá! Cẩn thận nhé.", "Uno… gần thắng rồi đấy!", "Còn 1 lá thôi — lo đi!", "Bạn sắp thua nếu không chặn tôi.", "Tôi gần về đích rồi."},
			[]string{"I have one card left! Careful.", "Uno… almost there!", "Just one — worry!", "You're in trouble.", "I'm nearly out."},
		)
		msg, _ := viOrEN(vi+" ("+seqVI+")", en+" ("+seqEN+")")
		return msg
	}

	if forcedDraw > 0 && forceCard != "" {
		forceVI, forceEN := unoSpeakCard(forceCard, "")
		vi, en := unoPick(
			[]string{
				"Nhận lấy " + fmt.Sprintf("%d", forcedDraw) + " lá nhé! Tôi đánh " + seqVI + ".",
				"Kế hoạch khá hay: " + forceVI + ". Bạn rút " + fmt.Sprintf("%d", forcedDraw) + ".",
				"Đây mới là bất ngờ — " + forceVI + "!",
				"Tôi đánh " + seqVI + ". Rút bài đi!",
				"Bạn thật xui: " + forceVI + ".",
			},
			[]string{
				"Take " + fmt.Sprintf("%d", forcedDraw) + " cards! I played " + seqEN + ".",
				"Nice plan: " + forceEN + ". Draw " + fmt.Sprintf("%d", forcedDraw) + ".",
				"Surprise — " + forceEN + "!",
				"I played " + seqEN + ". Draw!",
				"Unlucky: " + forceEN + ".",
			},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	}
	if len(played) > 1 {
		vi, en := unoPick(
			[]string{"Tôi đánh chuỗi: " + seqVI + ". Đến lượt bạn.", "Combo: " + seqVI + "!", "Tôi nghĩ đây là nước đi tốt: " + seqVI + ".", "Xong chuỗi. Lượt bạn."},
			[]string{"I played a combo: " + seqEN + ". Your turn.", "Combo: " + seqEN + "!", "Solid line: " + seqEN + ".", "Done. Your turn."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	}
	last := played[0]
	viName, enName := unoSpeakCard(last, "")
	var vi, en string
	switch {
	case strings.HasPrefix(last, "W"):
		vi, en = unoPick(
			[]string{"Tôi đổi màu. " + viName + ".", "Màu mới đây — tính lại nhé.", "Wild! Lượt bạn.", "Tôi chọn màu này."},
			[]string{"Color change. " + enName + ".", "New color — rethink.", "Wild! Your turn.", "I pick this color."},
		)
	case len(last) > 1 && last[1:] == "S":
		vi, en = unoPick(
			[]string{"Lượt của bạn bị bỏ! " + viName + ".", "Bạn vừa bị chặn.", "Skip — tôi đánh tiếp…", "Không đến lượt bạn đâu."},
			[]string{"You're skipped! " + enName + ".", "Blocked.", "Skip — I continue…", "Not your turn."},
		)
	case len(last) > 1 && last[1:] == "V":
		vi, en = unoPick(
			[]string{"Đổi chiều! " + viName + ".", "Mọi thứ đã thay đổi.", "Reverse — vòng xoay.", "Chiều chơi đảo lại."},
			[]string{"Reverse! " + enName + ".", "Everything flips.", "Direction changed.", "Spinning around."},
		)
	default:
		vi, en = unoPick(
			[]string{"Tôi chọn lá này: " + viName + ".", "Đến lượt bạn.", "Tôi nghĩ đây là nước đi tốt.", "Đánh " + viName + ".", "Lượt bạn nhé."},
			[]string{"I pick this: " + enName + ".", "Your turn.", "Solid move.", "Played " + enName + ".", "Go ahead."},
		)
	}
	msg, _ := viOrEN(vi, en)
	return msg
}

func (g *unoGame) snapshotLocked() map[string]interface{} {
	dir := "cw"
	if g.direction < 0 {
		dir = "ccw"
	}
	extra := map[string]interface{}{
		"hand":            append([]string{}, g.hand...),
		"top":             g.top,
		"botCount":        len(g.botHand),
		"color":           g.color,
		"direction":       dir,
		"deckCount":       len(g.deck),
		"pendingColor":    g.pendingColor,
		"pendingDraw4":    g.pendingDraw4,
		"humanLastCard":   g.humanLastCard,
		"botLastCard":     g.botLastCard,
		"botTurnCards":    append([]string{}, g.botTurnCards...),
		"botForcedDraw":   g.botForcedDraw,
		"botForceCard":    g.botForceCard,
		"drawAnimTo":      g.drawAnimTo,
		"drawAnimN":       g.drawAnimN,
		"botMustCont":     g.botMustCont,
		"saidUno":         g.saidUno,
		"justDrew":        g.justDrew,
		"botJustPlayed":   strings.HasPrefix(g.lastMove, "bot:"),
		"humanJustPlayed": strings.HasPrefix(g.lastMove, "human:") && !strings.HasSuffix(g.lastMove, ":draw") && !strings.HasSuffix(g.lastMove, ":pass") && !strings.HasSuffix(g.lastMove, ":uno"),
	}
	return g.baseSnap("uno", "uno", extra)
}

func (g *unoGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *unoGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	topVI, _ := unoSpeakCard(g.top, "")
	colVI := unoColorVI(g.color)
	return fmt.Sprintf("Uno. Bãi bài: %s. Màu đang chơi: %s. Bạn %d lá, robot %d lá. Trạng thái: %s.",
		topVI, colVI, len(g.hand), len(g.botHand), g.status)
}

func (g *unoGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	g.mu.Unlock()
	ng := newUnoGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	unoMu.Lock()
	unoInst = ng
	unoMu.Unlock()
	return ng.snapshot()
}

func (g *unoGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *unoGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *unoGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || !g.humanTurn || g.botThinking {
		return nil
	}
	if g.pendingColor {
		return []string{"color:R", "color:G", "color:B", "color:Y"}
	}
	var out []string
	for _, c := range g.hand {
		if g.canPlay(c) {
			out = append(out, "play:"+c)
		}
	}
	out = append(out, "draw", "pass")
	if len(g.hand) == 1 && !g.saidUno {
		out = append(out, "uno")
	}
	return out
}

func (g *unoGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	if g.botThinking {
		g.mu.Unlock()
		return nil, fmt.Errorf("robot đang đánh, vui lòng chờ")
	}
	cmd := strings.ToLower(strings.TrimSpace(uci))
	needBot := false
	var thinkGen int

	switch {
	case strings.HasPrefix(cmd, "play:"):
		if !g.humanTurn {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa đến lượt bạn")
		}
		if g.pendingColor {
			g.mu.Unlock()
			return nil, fmt.Errorf("cần chọn màu trước (color:R/G/B/Y)")
		}
		card := strings.ToUpper(strings.TrimPrefix(cmd, "play:"))
		idx := -1
		for i, c := range g.hand {
			if c == card {
				idx = i
				break
			}
		}
		if idx == -1 {
			g.mu.Unlock()
			return nil, fmt.Errorf("bạn không có lá %s", card)
		}
		if !g.canPlay(card) {
			g.mu.Unlock()
			return nil, fmt.Errorf("lá %s không hợp lệ trên lá %s", card, g.top)
		}
		g.hand = append(g.hand[:idx], g.hand[idx+1:]...)
		g.top = card
		g.discard = append(g.discard, card)
		g.lastMove = "human:" + card
		g.humanLastCard = card
		g.botForcedDraw = 0
		g.botForceCard = ""
		g.botTurnCards = nil
		g.drawAnimTo = ""
		g.drawAnimN = 0
		g.botMustCont = false
		g.justDrew = false
		g.moves++
		g.history = append(g.history, "H:"+card)

		if strings.HasPrefix(card, "W") {
			g.pendingColor = true
			g.pendingDraw4 = card == "W4"
			vi, en := unoPick(
				[]string{"Bạn đánh wild — chọn màu đi!", "Đổi màu nào?", "Wild! Màu đỏ? Xanh? Vàng?", "Chọn màu ở giữa màn hình.", "Tôi sẽ phải tính toán lại sau khi bạn chọn."},
				[]string{"Wild — pick a color!", "Change color?", "Wild! Red? Green? Blue?", "Pick a color in the center.", "I'll rethink after your color."},
			)
			g.message, _ = viOrEN(vi, en)
		} else {
			g.color = card[0:1]
			switch card[1:] {
			case "S":
				vi, en := unoPick(
					[]string{"Lượt của tôi bị bỏ!", "Bạn vừa chặn tôi.", "Skip — mạnh!", "Tôi mất lượt rồi.", "Nước đi hay."},
					[]string{"I'm skipped!", "You blocked me.", "Skip — strong!", "I lose a turn.", "Nice move."},
				)
				g.message, _ = viOrEN(vi, en)
			case "V":
				g.direction = -g.direction
				if g.direction == 0 {
					g.direction = 1
				}
				vi, en := unoPick(
					[]string{"Đổi chiều!", "Mọi thứ đã thay đổi.", "Reverse — vòng xoay.", "Bạn vừa đảo chiều.", "Nước đi hay."},
					[]string{"Reverse!", "Everything flipped.", "Direction changed.", "You reversed it.", "Nice move."},
				)
				g.message, _ = viOrEN(vi, en)
			case "D":
				g.drawNLocked(&g.botHand, 2)
				g.drawAnimTo = "bot"
				g.drawAnimN = 2
				g.botForceCard = card
				g.botForcedDraw = 0
				vi, en := unoPick(
					[]string{"Nhận lấy hai lá nhé… với tôi.", "Cộng hai! Tôi đang rút.", "Kế hoạch khá hay.", "Bạn tàn nhẫn đấy.", "Tôi phải rút hai lá."},
					[]string{"Take two… from me.", "Draw two! I'm drawing.", "Nice plan.", "Ruthless.", "I draw two."},
				)
				g.message, _ = viOrEN(vi, en)
			default:
				g.humanTurn = false
				vi, en := unoPick(
					[]string{"Nước đi hay.", "Tôi đã đoán trước.", "Bạn đang chiếm ưu thế.", "Ổn. Đến lượt tôi.", "Được đấy."},
					[]string{"Nice move.", "I saw that coming.", "You're ahead.", "Okay. My turn.", "Solid."},
				)
				g.message, _ = viOrEN(vi, en)
			}
		}

		if len(g.hand) == 1 && !g.saidUno {
			extraVI, extraEN := unoPick(
				[]string{" Đừng quên hô UNO!", " Bạn còn một lá — hô UNO đi!", " Gần thắng rồi, nhớ UNO!"},
				[]string{" Don't forget UNO!", " One card left — call UNO!", " Almost — shout UNO!"},
			)
			g.message += extraVI
			_ = extraEN
		}

		if len(g.hand) == 0 {
			g.status = "win"
			g.winner = "human"
			g.pendingColor = false
			g.pendingDraw4 = false
			g.message = unoWinMessage(true)
			g.humanTurn = false
		} else if !g.humanTurn && !g.pendingColor {
			needBot = true
		}

	case cmd == "draw":
		if !g.humanTurn {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa đến lượt bạn")
		}
		if g.pendingColor {
			g.mu.Unlock()
			return nil, fmt.Errorf("cần chọn màu trước")
		}
		if c := g.drawCardLocked(); c != "" {
			g.hand = append(g.hand, c)
			g.lastMove = "human:draw"
			g.drawAnimTo = "human"
			g.drawAnimN = 1
			g.justDrew = true
			g.saidUno = false // drew more cards
			vi, en := unoPick(
				[]string{"Bạn không còn lựa chọn — rút đi.", "Có vẻ hơi xui.", "Rút một lá. Đánh được không?", "Bộ bài thêm một lá.", "Xui một chút nhé."},
				[]string{"No choice — draw.", "Unlucky.", "Drew one. Can you play?", "One more card.", "A bit unlucky."},
			)
			g.message, _ = viOrEN(vi, en)
		} else {
			g.message, _ = viOrEN("Bộ bài đã hết.", "The deck is empty.")
		}

	case cmd == "pass":
		if !g.humanTurn {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa đến lượt bạn")
		}
		if g.pendingColor {
			g.mu.Unlock()
			return nil, fmt.Errorf("cần chọn màu trước")
		}
		g.humanTurn = false
		g.justDrew = false
		g.drawAnimTo = ""
		g.drawAnimN = 0
		g.lastMove = "human:pass"
		vi, en := unoPick(
			[]string{"Bạn bỏ lượt. Tới tôi.", "Pass — tôi đánh nhé.", "Không đánh được à? Được rồi."},
			[]string{"You pass. My turn.", "Pass — my go.", "Can't play? Okay."},
		)
		g.message, _ = viOrEN(vi, en)
		needBot = true

	case cmd == "uno":
		if !g.humanTurn || len(g.hand) != 1 {
			g.mu.Unlock()
			return nil, fmt.Errorf("chỉ hô UNO khi còn 1 lá")
		}
		g.saidUno = true
		g.lastMove = "human:uno"
		vi, en := unoPick(
			[]string{"UNO! Tôi nghe rồi.", "Bạn sắp thắng rồi!", "Đừng quên… bạn đã hô UNO.", "UNO — nguy hiểm quá!", "Tốt, bạn đã hô UNO."},
			[]string{"UNO! Heard you.", "You're about to win!", "Good call — UNO.", "UNO — dangerous!", "Nice UNO shout."},
		)
		g.message, _ = viOrEN(vi, en)

	case strings.HasPrefix(cmd, "color:"):
		if !g.pendingColor {
			g.mu.Unlock()
			return nil, fmt.Errorf("không có lá wild nào đang chờ chọn màu")
		}
		col := strings.ToUpper(strings.TrimPrefix(cmd, "color:"))
		if col != "R" && col != "G" && col != "B" && col != "Y" {
			g.mu.Unlock()
			return nil, fmt.Errorf("màu không hợp lệ (R/G/B/Y)")
		}
		g.color = col
		g.pendingColor = false
		colVI, colEN := unoColorVI(col), unoColorEN(col)
		if g.pendingDraw4 {
			g.drawNLocked(&g.botHand, 4)
			g.pendingDraw4 = false
			g.drawAnimTo = "bot"
			g.drawAnimN = 4
			g.botForceCard = g.humanLastCard
			g.botForcedDraw = 0
			vi, en := unoPick(
				[]string{"Bạn đổi sang màu " + colVI + ". Tôi rút bốn lá…", "Màu " + colVI + " và +4 — tàn nhẫn!", "Đây mới là bất ngờ. Tôi rút 4.", "Tôi phải rút bốn lá. Màu " + colVI + "."},
				[]string{"You chose " + colEN + ". I draw four…", colEN + " and +4 — ruthless!", "Surprise. Drawing four.", "Four cards. Color " + colEN + "."},
			)
			g.message, _ = viOrEN(vi, en)
		} else {
			g.humanTurn = false
			vi, en := unoPick(
				[]string{"Bạn đổi sang màu " + colVI + ".", "Màu " + colVI + " sao? Tôi tính lại.", "Màu mới: " + colVI + ". Lượt tôi.", "Tôi sẽ phải tính toán lại."},
				[]string{"You chose " + colEN + ".", colEN + "? I'll recalculate.", "New color: " + colEN + ". My turn.", "I need to rethink."},
			)
			g.message, _ = viOrEN(vi, en)
			needBot = true
		}

	default:
		g.mu.Unlock()
		return nil, fmt.Errorf("lệnh không hợp lệ (play:<card>/draw/pass/uno/color:<X>)")
	}

	if needBot && g.status == "playing" {
		g.botThinking = true
		g.thinkGen++
		thinkGen = g.thinkGen
	}

	resp := g.snapshotLocked()
	msg := g.message
	winner := g.winner
	status := g.status
	g.mu.Unlock()

	speakHint := "Uno"
	if status == "win" && winner == "human" {
		speakHint = "Uno. Chúc mừng! Người chơi thắng."
	} else if status == "win" && winner == "bot" {
		speakHint = "Uno. Robot thắng."
	}
	queueGameSpeak("uno", msg, speakHint)
	if needBot {
		go g.runBotThink(thinkGen)
	}
	return resp, nil
}

func (g *unoGame) runBotThink(gen int) {
	// Give the human play animation ~2s before the robot responds.
	time.Sleep(2 * time.Second)

	// Clear combo trackers at the start of this think.
	g.mu.Lock()
	if gen != g.thinkGen || !g.botThinking {
		g.mu.Unlock()
		return
	}
	g.botForcedDraw = 0
	g.botForceCard = ""
	g.botTurnCards = nil
	g.botMustCont = false
	g.mu.Unlock()

	for step := 0; step < 20; step++ {
		g.mu.Lock()
		if gen != g.thinkGen || !g.botThinking {
			g.mu.Unlock()
			return
		}
		if g.status != "playing" || g.humanTurn {
			g.botThinking = false
			msg := g.message
			g.mu.Unlock()
			if msg != "" {
				queueGameSpeak("uno", msg, "Uno")
			}
			return
		}

		result := g.botPlayOneStepLocked()
		msg := g.message
		card := g.botLastCard
		col := g.color
		winner := g.winner
		status := g.status
		drawN := g.drawAnimN
		g.mu.Unlock()

		if msg != "" {
			hint := "Uno. Robot: " + unoSpeakCardMode(card, "")
			if strings.HasPrefix(card, "W") {
				hint = "Uno. Robot: " + unoSpeakCardMode(card, col)
			}
			if status == "win" && winner == "bot" {
				hint = "Uno. Robot thắng."
			}
			if drawN > 0 {
				hint = msg
			}
			queueGameSpeak("uno", msg, hint)
		}

		if result == "done" {
			g.mu.Lock()
			g.botThinking = false
			g.drawAnimTo = ""
			g.drawAnimN = 0
			g.botMustCont = false
			g.mu.Unlock()
			return
		}

		// Keep +2/+4 (or skip) on the table while draw/skip anim plays, then continue.
		if result == "draw_pause" {
			time.Sleep(2200 * time.Millisecond)
		} else {
			time.Sleep(900 * time.Millisecond)
		}
	}

	g.mu.Lock()
	g.botThinking = false
	g.humanTurn = true
	g.mu.Unlock()
}

// NewUno builds the simplified 2-player Uno minigame mod.
func NewUno() *genericBoardMod {
	return newGenericBoardMod("Uno", "uno",
		"Uno đơn giản 2 người — bạn đấu với bot",
		func() string { return getUno().summaryText() },
		func() map[string]interface{} { return getUno().snapshot() },
		func() map[string]interface{} { return getUno().reset() },
		func(s string) string { return getUno().setDifficulty(s) },
		func() string { return getUno().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getUno().playUCI(u) },
		func() []string { return getUno().legalUCIs() },
		func() (string, string) {
			vi, en := unoPick(
				[]string{"Hãy bắt đầu nào!", "Tôi sẽ cố gắng thắng.", "Tôi chia bài nhé.", "Uno mới — chúc may mắn!", "Sẵn sàng chưa? Bắt đầu!"},
				[]string{"Let's begin!", "I'll try hard to win.", "Dealing the cards.", "New Uno — good luck!", "Ready? Let's start!"},
			)
			say, _ := viOrEN(vi, en)
			return say, "Uno. Ván mới."
		},
	)
}
