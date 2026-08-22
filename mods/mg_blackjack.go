package mods

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Web Blackjack vs a simple dealer bot. UCI: "deal", "hit", "stand".
// Dealer draws one card at a time (with pause + speak) instead of all at once.

func bjNewDeck() []string {
	suits := []string{"S", "H", "D", "C"}
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	var deck []string
	for _, s := range suits {
		for _, r := range ranks {
			deck = append(deck, r+s)
		}
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

// bjCardRank extracts the rank portion of a "<rank><suit>" card, e.g. "10S" -> "10".
func bjCardRank(card string) string {
	if len(card) >= 3 && card[0] == '1' && card[1] == '0' {
		return "10"
	}
	if len(card) == 0 {
		return ""
	}
	return card[:1]
}

func bjCardValue(card string) int {
	switch r := bjCardRank(card); r {
	case "A":
		return 11
	case "J", "Q", "K", "10":
		return 10
	default:
		n, _ := strconv.Atoi(r)
		return n
	}
}

// bjHandValue sums a hand, softening aces (11 -> 1) while over 21.
func bjHandValue(hand []string) int {
	total, aces := 0, 0
	for _, c := range hand {
		total += bjCardValue(c)
		if bjCardRank(c) == "A" {
			aces++
		}
	}
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}
	return total
}

func bjSpeakRank(card string) (vi, en string) {
	switch r := bjCardRank(card); r {
	case "A":
		return "Át", "Ace"
	case "J":
		return "J", "Jack"
	case "Q":
		return "Q", "Queen"
	case "K":
		return "K", "King"
	default:
		return r, r
	}
}

func bjSpeakCardMode(card string) string {
	lang := gameSpeakLang()
	if lang == "en" {
		_, en := bjSpeakRank(card)
		return en
	}
	if lang == "vi" {
		vi, _ := bjSpeakRank(card)
		return vi
	}
	return bjRankSpoken(card)
}

type bjGame struct {
	miniCommon
	deck           []string
	playerHand     []string
	dealerHand     []string
	dealt          bool
	dealerRevealed bool
	bank           int
	lastCard       string
	drawAnimTo     string // "player" | "dealer" | ""
}

var (
	bjMu   sync.Mutex
	bjInst *bjGame
)

func getBlackjackWeb() *bjGame {
	bjMu.Lock()
	defer bjMu.Unlock()
	if bjInst == nil {
		bjInst = newBjGame()
	}
	return bjInst
}

func newBjGame() *bjGame {
	g := &bjGame{bank: 100}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	g.message, _ = viOrEN("Ván Blackjack mới. Bấm Bắt đầu để chia bài.", "New Blackjack round. Press Start to deal.")
	return g
}

func (g *bjGame) drawLocked() string {
	if len(g.deck) == 0 {
		g.deck = bjNewDeck()
	}
	c := g.deck[len(g.deck)-1]
	g.deck = g.deck[:len(g.deck)-1]
	return c
}

func (g *bjGame) askHitMsg(justCard string) string {
	pv := bjHandValue(g.playerHand)
	if justCard != "" {
		msg, _ := speakf(
			"Bạn đã rút được lá %s. Tổng điểm hiện tại là %d. Bạn có muốn rút thêm không?",
			"You drew a %s. Your total is %d. Do you want to hit again?",
			bjSpeakCardMode(justCard), pv,
		)
		return msg
	}
	msg, _ := speakf(
		"Điểm hiện tại của bạn là %d. Bạn có muốn rút thêm không?",
		"Your current score is %d. Do you want to hit?",
		pv,
	)
	return msg
}

func (g *bjGame) finishResolveLocked() {
	pv, dv := bjHandValue(g.playerHand), bjHandValue(g.dealerHand)
	switch {
	case pv > 21:
		g.status, g.winner, g.bank = "lose", "bot", g.bank-10
		g.message, _ = viOrEN("Quắc! Bạn thua.", "Bust! You lose.")
	case dv > 21:
		g.status, g.winner, g.bank = "win", "human", g.bank+10
		g.message, _ = speakf(
			"Nhà cái quắc với %d điểm! Bạn thắng!",
			"Dealer busts with %d! You win!",
			dv,
		)
	case pv > dv:
		g.status, g.winner, g.bank = "win", "human", g.bank+10
		g.message, _ = speakf(
			"Bạn %d — nhà cái %d. Bạn thắng!",
			"You %d — dealer %d. You win!",
			pv, dv,
		)
	case pv < dv:
		g.status, g.winner, g.bank = "lose", "bot", g.bank-10
		g.message, _ = speakf(
			"Bạn %d — nhà cái %d. Bạn thua.",
			"You %d — dealer %d. You lose.",
			pv, dv,
		)
	default:
		g.status, g.winner = "draw", ""
		g.message, _ = speakf(
			"Hoà (push) ở %d điểm.",
			"Push at %d.",
			pv,
		)
	}
	g.humanTurn = false
	g.drawAnimTo = ""
}

// beginDealerTurnLocked flips to dealer phase (hole card revealed) and
// schedules stepped draws via runDealerThink.
func (g *bjGame) beginDealerTurnLocked() int {
	g.humanTurn = false
	g.dealerRevealed = true
	g.botThinking = true
	g.thinkGen++
	g.drawAnimTo = ""
	g.lastCard = ""
	dv := bjHandValue(g.dealerHand)
	g.message, _ = speakf(
		"Đủ rồi — tới lượt nhà cái. Nhà cái đang có %d điểm.",
		"Standing — dealer's turn. Dealer shows %d.",
		dv,
	)
	return g.thinkGen
}

func (g *bjGame) snapshotLocked() map[string]interface{} {
	dealerShown := append([]string{}, g.dealerHand...)
	dealerValue := bjHandValue(g.dealerHand)
	if g.dealt && g.status == "playing" && !g.dealerRevealed && len(dealerShown) > 0 {
		dealerShown = []string{dealerShown[0], "??"}
		dealerValue = bjCardValue(g.dealerHand[0])
	}
	extra := map[string]interface{}{
		"playerHand":     append([]string{}, g.playerHand...),
		"dealerHand":     dealerShown,
		"playerValue":    bjHandValue(g.playerHand),
		"dealerValue":    dealerValue,
		"bank":           g.bank,
		"dealt":          g.dealt,
		"dealerRevealed": g.dealerRevealed,
		"lastCard":       g.lastCard,
		"drawAnimTo":     g.drawAnimTo,
	}
	return g.baseSnap("blackjack", "cards", extra)
}

func (g *bjGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *bjGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Blackjack web. Đã chia bài: %v. Điểm người chơi: %d. Trạng thái: %s. Bank: %d.",
		g.dealt, bjHandValue(g.playerHand), g.status, g.bank)
}

func (g *bjGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff, prevBank := g.difficulty, g.bank
	g.mu.Unlock()
	ng := newBjGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	ng.bank = prevBank
	bjMu.Lock()
	bjInst = ng
	bjMu.Unlock()
	return ng.snapshot()
}

func (g *bjGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *bjGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *bjGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.botThinking {
		return nil
	}
	if !g.dealt || g.status != "playing" {
		return []string{"deal"}
	}
	if !g.humanTurn {
		return nil
	}
	return []string{"hit", "stand"}
}

func (g *bjGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.botThinking {
		g.mu.Unlock()
		return nil, fmt.Errorf("nhà cái đang rút bài, vui lòng chờ")
	}
	cmd := strings.ToLower(strings.TrimSpace(uci))
	needDealer := false
	var thinkGen int

	switch cmd {
	case "deal":
		g.deck = bjNewDeck()
		g.playerHand = []string{g.drawLocked(), g.drawLocked()}
		g.dealerHand = []string{g.drawLocked(), g.drawLocked()}
		g.status = "playing"
		g.winner = ""
		g.humanTurn = true
		g.dealt = true
		g.dealerRevealed = false
		g.drawAnimTo = "player"
		g.lastCard = g.playerHand[len(g.playerHand)-1]
		g.moves++
		g.lastMove = "deal"
		pv := bjHandValue(g.playerHand)
		if pv == 21 {
			// Natural / soft 21 — go straight to dealer.
			thinkGen = g.beginDealerTurnLocked()
			needDealer = true
		} else {
			g.message = g.askHitMsg("")
		}

	case "hit":
		if !g.dealt || g.status != "playing" || !g.humanTurn {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa đến lượt rút của bạn")
		}
		card := g.drawLocked()
		g.playerHand = append(g.playerHand, card)
		g.lastCard = card
		g.drawAnimTo = "player"
		g.lastMove = "hit"
		g.moves++
		pv := bjHandValue(g.playerHand)
		if pv > 21 {
			g.status, g.winner, g.bank = "lose", "bot", g.bank-10
			g.humanTurn = false
			g.dealerRevealed = true
			g.message, _ = speakf(
				"Bạn rút lá %s — tổng %d. Quắc! Bạn thua.",
				"You drew a %s — total %d. Bust! You lose.",
				bjSpeakCardMode(card), pv,
			)
		} else if len(g.playerHand) >= 5 {
			g.status, g.winner, g.bank = "win", "human", g.bank+10
			g.humanTurn = false
			g.dealerRevealed = true
			g.message, _ = viOrEN("Năm lá Charlie! Bạn thắng!", "Five-card Charlie! You win!")
		} else if pv == 21 {
			thinkGen = g.beginDealerTurnLocked()
			needDealer = true
		} else {
			g.message = g.askHitMsg(card)
		}

	case "stand":
		if !g.dealt || g.status != "playing" || !g.humanTurn {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa đến lượt của bạn")
		}
		g.lastMove = "stand"
		g.moves++
		g.drawAnimTo = ""
		g.lastCard = ""
		thinkGen = g.beginDealerTurnLocked()
		needDealer = true

	default:
		g.mu.Unlock()
		return nil, fmt.Errorf("lệnh không hợp lệ (deal/hit/stand)")
	}

	resp := g.snapshotLocked()
	msg := g.message
	winner := g.winner
	status := g.status
	g.mu.Unlock()

	hint := "Blackjack"
	if status == "win" && winner == "human" {
		hint = "Blackjack. Chúc mừng! Bạn thắng."
	} else if status == "lose" {
		hint = "Blackjack. Bạn thua."
	}
	queueGameSpeak("blackjack", msg, hint)
	if needDealer {
		go g.runDealerThink(thinkGen)
	}
	return resp, nil
}

func (g *bjGame) runDealerThink(gen int) {
	// Let the stand / 21 message + player anim settle first.
	time.Sleep(1600 * time.Millisecond)

	for step := 0; step < 12; step++ {
		g.mu.Lock()
		if gen != g.thinkGen || !g.botThinking {
			g.mu.Unlock()
			return
		}
		if g.status != "playing" {
			g.botThinking = false
			g.mu.Unlock()
			return
		}

		dv := bjHandValue(g.dealerHand)
		if dv >= 17 {
			g.finishResolveLocked()
			g.botThinking = false
			msg := g.message
			winner := g.winner
			status := g.status
			g.mu.Unlock()
			hint := "Blackjack"
			if status == "win" && winner == "human" {
				hint = "Blackjack. Chúc mừng! Bạn thắng."
			} else if status == "lose" {
				hint = "Blackjack. Bạn thua."
			}
			queueGameSpeak("blackjack", msg, hint)
			return
		}

		card := g.drawLocked()
		g.dealerHand = append(g.dealerHand, card)
		g.lastCard = card
		g.drawAnimTo = "dealer"
		g.lastMove = "dealer_hit"
		dv = bjHandValue(g.dealerHand)
		rank := bjSpeakCardMode(card)
		if dv > 21 {
			g.message, _ = speakf(
				"Nhà cái rút lá %s. Tổng nhà cái %d — quắc!",
				"Dealer draws a %s. Dealer total %d — bust!",
				rank, dv,
			)
		} else if dv >= 17 {
			g.message, _ = speakf(
				"Nhà cái rút lá %s. Tổng nhà cái %d — dừng rút.",
				"Dealer draws a %s. Dealer total %d — stands.",
				rank, dv,
			)
		} else {
			g.message, _ = speakf(
				"Nhà cái rút lá %s. Tổng nhà cái hiện tại %d — rút tiếp.",
				"Dealer draws a %s. Dealer total is %d — hits again.",
				rank, dv,
			)
		}
		msg := g.message
		g.mu.Unlock()

		queueGameSpeak("blackjack", msg, "Blackjack. Nhà cái: "+bjSpeakCardMode(card))
		time.Sleep(2000 * time.Millisecond)
	}

	g.mu.Lock()
	if gen == g.thinkGen && g.botThinking && g.status == "playing" {
		g.finishResolveLocked()
		g.botThinking = false
		msg := g.message
		g.mu.Unlock()
		queueGameSpeak("blackjack", msg, "Blackjack")
		return
	}
	g.mu.Unlock()
}

// NewBlackjackWeb builds the web Blackjack minigame mod.
func NewBlackjackWeb() *genericBoardMod {
	return newGenericBoardMod("BlackjackWeb", "bjweb",
		"Blackjack web — đấu với dealer bot",
		func() string { return getBlackjackWeb().summaryText() },
		func() map[string]interface{} { return getBlackjackWeb().snapshot() },
		func() map[string]interface{} { return getBlackjackWeb().reset() },
		func(s string) string { return getBlackjackWeb().setDifficulty(s) },
		func() string { return getBlackjackWeb().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getBlackjackWeb().playUCI(u) },
		func() []string { return getBlackjackWeb().legalUCIs() },
		func() (string, string) {
			say, _ := viOrEN("Ván Blackjack mới. Bấm Bắt đầu để chia bài.", "New Blackjack round. Press Start to deal.")
			return say, "Blackjack. Ván mới."
		},
	)
}
