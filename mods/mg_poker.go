package mods

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Simplified 5-card draw poker vs bot: deal 5 each, human may discard/redraw
// once ("draw:1,3"), bot auto-draws, then "show" compares hand ranks.

var pokerRankOrder = map[string]int{
	"2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7, "8": 8, "9": 9, "10": 10,
	"J": 11, "Q": 12, "K": 13, "A": 14,
}

func pokerNewDeck() []string {
	suits := []string{"S", "H", "D", "C"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	var deck []string
	for _, s := range suits {
		for _, r := range ranks {
			deck = append(deck, r+s)
		}
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

func pokerCardRank(card string) string {
	if len(card) >= 3 && card[0] == '1' && card[1] == '0' {
		return "10"
	}
	if len(card) == 0 {
		return ""
	}
	return card[:1]
}

func pokerCardSuit(card string) string {
	if len(card) >= 3 && card[0] == '1' && card[1] == '0' {
		return card[2:]
	}
	if len(card) < 2 {
		return ""
	}
	return card[1:]
}

func pokerRankVal(rank string) int {
	return pokerRankOrder[rank]
}

func pokerPick(viLines, enLines []string) (vi, en string) {
	if len(viLines) == 0 {
		return "", ""
	}
	i := rand.Intn(len(viLines))
	vi = viLines[i]
	if i < len(enLines) {
		en = enLines[i]
	} else {
		en = vi
	}
	return
}

// pokerEvaluate5 scores a 5-card hand.
// category 0=high .. 8=straight flush. key is stable for UI/i18n.
func pokerEvaluate5(hand []string) (category int, key, nameVI, nameEN string, tiebreak []int) {
	ranks := make([]int, 5)
	suits := make([]string, 5)
	for i, c := range hand {
		ranks[i] = pokerRankVal(pokerCardRank(c))
		suits[i] = pokerCardSuit(c)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ranks)))

	flush := true
	for i := 1; i < 5; i++ {
		if suits[i] != suits[0] {
			flush = false
		}
	}

	dedup := []int{}
	for _, r := range ranks {
		if len(dedup) == 0 || dedup[len(dedup)-1] != r {
			dedup = append(dedup, r)
		}
	}
	straight, straightHigh := false, 0
	if len(dedup) == 5 {
		if dedup[0]-dedup[4] == 4 {
			straight, straightHigh = true, dedup[0]
		} else if dedup[0] == 14 && dedup[1] == 5 && dedup[2] == 4 && dedup[3] == 3 && dedup[4] == 2 {
			straight, straightHigh = true, 5
		}
	}

	counts := map[int]int{}
	for _, r := range ranks {
		counts[r]++
	}
	type rankCount struct{ rank, count int }
	var groups []rankCount
	for r, c := range counts {
		groups = append(groups, rankCount{r, c})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].count != groups[j].count {
			return groups[i].count > groups[j].count
		}
		return groups[i].rank > groups[j].rank
	})

	switch {
	case straight && flush:
		return 8, "straight_flush", "THÙNG PHÁ SẢNH", "STRAIGHT FLUSH", []int{straightHigh}
	case groups[0].count == 4:
		return 7, "four", "TỨ QUÝ", "FOUR OF A KIND", []int{groups[0].rank, groups[1].rank}
	case groups[0].count == 3 && len(groups) > 1 && groups[1].count == 2:
		return 6, "full_house", "CÙ LŨ", "FULL HOUSE", []int{groups[0].rank, groups[1].rank}
	case flush:
		return 5, "flush", "THÙNG", "FLUSH", ranks
	case straight:
		return 4, "straight", "SẢNH", "STRAIGHT", []int{straightHigh}
	case groups[0].count == 3:
		return 3, "three", "BỘ BA", "THREE OF A KIND", []int{groups[0].rank, groups[1].rank, groups[2].rank}
	case groups[0].count == 2 && len(groups) > 1 && groups[1].count == 2:
		hi, lo := groups[0].rank, groups[1].rank
		if lo > hi {
			hi, lo = lo, hi
		}
		return 2, "two_pair", "HAI ĐÔI", "TWO PAIR", []int{hi, lo, groups[2].rank}
	case groups[0].count == 2:
		tb := []int{groups[0].rank}
		for _, g := range groups[1:] {
			tb = append(tb, g.rank)
		}
		return 1, "pair", "MỘT ĐÔI", "ONE PAIR", tb
	default:
		return 0, "high", "MẬU THẦU", "HIGH CARD", ranks
	}
}

func pokerCompareHands(catA int, tbA []int, catB int, tbB []int) int {
	if catA != catB {
		if catA > catB {
			return 1
		}
		return -1
	}
	n := len(tbA)
	if len(tbB) < n {
		n = len(tbB)
	}
	for i := 0; i < n; i++ {
		if tbA[i] != tbB[i] {
			if tbA[i] > tbB[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func pokerBotDrawIndices(hand []string) []int {
	counts := map[string]int{}
	for _, c := range hand {
		counts[pokerCardRank(c)]++
	}
	var discard []int
	for i, c := range hand {
		r := pokerCardRank(c)
		if counts[r] >= 2 || pokerRankVal(r) >= 11 {
			continue
		}
		discard = append(discard, i)
	}
	if len(discard) > 3 {
		discard = discard[:3]
	}
	return discard
}

type pokerHistEntry struct {
	Round  int    `json:"round"`
	Hand   string `json:"hand"`
	Result string `json:"result"` // win | lose | draw
}

type pokerGame struct {
	miniCommon
	deck       []string
	playerHand []string
	botHand    []string
	drawn      bool
	dealt      bool
	drawCount  int
	roundN     int
	history    []pokerHistEntry
	playerCat  int
	playerKey  string
	playerName string
	botCat     int
	botKey     string
	botName    string
}

var (
	pokerMu   sync.Mutex
	pokerInst *pokerGame
)

func getPoker() *pokerGame {
	pokerMu.Lock()
	defer pokerMu.Unlock()
	if pokerInst == nil {
		pokerInst = newPokerGame()
	}
	return pokerInst
}

func newPokerGame() *pokerGame {
	g := &pokerGame{history: nil}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	vi, en := pokerPick(
		[]string{
			"Tôi sẽ chia bài cho bạn.",
			"Chúc bạn may mắn.",
			"Hy vọng hôm nay vận may đứng về phía bạn.",
			"Đừng quên chọn những lá bài cần đổi.",
		},
		[]string{
			"I'll deal you in.",
			"Good luck.",
			"Hope luck is on your side today.",
			"Don't forget which cards to redraw.",
		},
	)
	g.message, _ = viOrEN(vi, en)
	return g
}

func (g *pokerGame) refreshEvalLocked() {
	if len(g.playerHand) == 5 {
		cat, key, vi, en, _ := pokerEvaluate5(g.playerHand)
		g.playerCat, g.playerKey = cat, key
		if getChessCommentMode() == chessModeGoogleVI {
			g.playerName = vi
		} else {
			g.playerName = en
		}
	}
	if len(g.botHand) == 5 {
		cat, key, vi, en, _ := pokerEvaluate5(g.botHand)
		g.botCat, g.botKey = cat, key
		if getChessCommentMode() == chessModeGoogleVI {
			g.botName = vi
		} else {
			g.botName = en
		}
	}
}

func (g *pokerGame) commentAfterDeal() string {
	g.refreshEvalLocked()
	name := g.playerName
	switch g.playerCat {
	case 0:
		vi, en := pokerPick(
			[]string{"Bạn đang có mậu thầu. Có vài lá nên đổi.", "Bộ bài này còn yếu — hãy chọn lá để đổi.", "Có tiềm năng nếu đổi đúng lá."},
			[]string{"Just a high card. Some cards should go.", "This hand is weak — pick cards to redraw.", "There's potential if you redraw well."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	case 1:
		vi, en := pokerPick(
			[]string{"Bạn đã có một đôi. Tôi thấy bộ bài này có tiềm năng.", "Một đôi là khởi đầu tốt. Nếu là tôi, tôi sẽ giữ đôi đó.", "Bạn vừa có một đôi — khá ổn."},
			[]string{"You've got a pair. This hand has potential.", "A pair is a solid start — I'd keep it.", "One pair already — not bad."},
		)
		msg, _ := viOrEN(vi+" ("+name+")", en+" ("+name+")")
		return msg
	case 2:
		vi, en := pokerPick(
			[]string{"Hai đôi rồi! Bộ bài khá mạnh.", "Bạn đang đi đúng hướng với hai đôi."},
			[]string{"Two pair already! Strong start.", "You're on the right track with two pair."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	default:
		vi, en := pokerPick(
			[]string{"Wow — "+name+" ngay từ đầu!", "Bộ bài đang rất đẹp: "+name+".", "Tôi sẽ giữ nguyên nếu là bạn."},
			[]string{"Wow — "+name+" right away!", "Beautiful hand: "+name+".", "I'd stand pat if I were you."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	}
}

func (g *pokerGame) commentAfterDraw() string {
	g.refreshEvalLocked()
	switch g.playerCat {
	case 0:
		vi, en := pokerPick(
			[]string{"Chưa được như mong đợi.", "Vẫn là mậu thầu — không sao.", "Lần sau sẽ tốt hơn."},
			[]string{"Not what we hoped for.", "Still high card — that's okay.", "Next time will be better."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	case 1, 2:
		vi, en := pokerPick(
			[]string{"Tốt hơn rồi. Bạn vừa có "+g.playerName+".", "Bộ bài đang mạnh dần.", "Có tiềm năng chiến thắng."},
			[]string{"Better! You've got "+g.playerName+".", "The hand is getting stronger.", "There's a chance to win."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	default:
		vi, en := pokerPick(
			[]string{"Rất đẹp! "+g.playerName+"!", "Bộ bài cực mạnh sau khi đổi.", "Khả năng thắng rất cao."},
			[]string{"Gorgeous! "+g.playerName+"!", "Huge hand after the redraw.", "You're looking like a favorite."},
		)
		msg, _ := viOrEN(vi, en)
		return msg
	}
}

func (g *pokerGame) commentShow(result int) string {
	g.refreshEvalLocked()
	handLineVI := "Đây là " + g.playerName + "."
	handLineEN := "That's " + g.playerName + "."
	var extraVI, extraEN string
	switch {
	case result > 0 && g.playerCat >= 6:
		extraVI, extraEN = pokerPick(
			[]string{"Xuất sắc! Bạn đã chiến thắng.", "Tuyệt vời! Bộ bài thật ấn tượng.", "Tôi biết bạn sẽ làm được.", "Chúc mừng! Bạn vừa tạo được "+g.playerName+".", "Bạn thật sự rất may mắn hôm nay."},
			[]string{"Brilliant! You win.", "Amazing hand!", "I knew you could do it.", "Congrats on that "+g.playerName+"!", "Luck is really on your side."},
		)
	case result > 0:
		extraVI, extraEN = pokerPick(
			[]string{"Bạn thắng với "+g.playerName+".", "Chúc mừng chiến thắng!", "Kết quả rất đẹp.", "Chơi thêm một ván nữa nhé."},
			[]string{"You win with "+g.playerName+".", "Congratulations!", "Nice result.", "Let's play another."},
		)
	case result < 0:
		extraVI, extraEN = pokerPick(
			[]string{"Không sao. Lần sau chúng ta sẽ thắng.", "Chỉ thiếu một chút may mắn.", "Robot thắng với "+g.botName+".", "Tôi tin bạn sẽ có bộ bài đẹp hơn."},
			[]string{"No worries — next time.", "Just a bit of luck short.", "I win with "+g.botName+".", "You'll get a better hand."},
		)
	default:
		extraVI, extraEN = pokerPick(
			[]string{"Hoà! Cả hai đều có "+g.playerName+".", "Push — cân sức."},
			[]string{"Push! Both have "+g.playerName+".", "It's a tie."},
		)
	}
	msg, _ := viOrEN(handLineVI+" "+extraVI, handLineEN+" "+extraEN)
	return msg
}

func (g *pokerGame) snapshotLocked() map[string]interface{} {
	botShown := make([]string, len(g.botHand))
	for i := range botShown {
		botShown[i] = "??"
	}
	if g.status != "playing" {
		copy(botShown, g.botHand)
	}
	phase := "idle"
	if g.dealt && g.status == "playing" && !g.drawn {
		phase = "select"
	} else if g.dealt && g.status == "playing" && g.drawn {
		phase = "ready_show"
	} else if g.status != "playing" {
		phase = "showdown"
	}
	hist := make([]map[string]interface{}, 0, len(g.history))
	for _, h := range g.history {
		hist = append(hist, map[string]interface{}{
			"round": h.Round, "hand": h.Hand, "result": h.Result,
		})
	}
	extra := map[string]interface{}{
		"playerHand":   append([]string{}, g.playerHand...),
		"botHand":      botShown,
		"drawn":        g.drawn,
		"dealt":        g.dealt,
		"phase":        phase,
		"drawCount":    g.drawCount,
		"playerCat":    g.playerCat,
		"playerCatKey": g.playerKey,
		"playerRank":   g.playerName,
		"history":      hist,
	}
	if g.status != "playing" {
		extra["botCat"] = g.botCat
		extra["botCatKey"] = g.botKey
		extra["botRank"] = g.botName
		score := 100 + g.playerCat*120 - g.drawCount*20
		if score < 50 {
			score = 50
		}
		if g.winner != "human" {
			score = 0
		}
		extra["score"] = score
	}
	return g.baseSnap("poker", "cards", extra)
}

func (g *pokerGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *pokerGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("Poker rút bài 5 lá. Đã chia: %v. Đã đổi bài: %v. Trạng thái: %s.",
		g.dealt, g.drawn, g.status)
}

func (g *pokerGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	prevHist := append([]pokerHistEntry{}, g.history...)
	prevRound := g.roundN
	g.mu.Unlock()
	ng := newPokerGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	ng.history = prevHist
	ng.roundN = prevRound
	pokerMu.Lock()
	pokerInst = ng
	pokerMu.Unlock()
	return ng.snapshot()
}

func (g *pokerGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *pokerGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *pokerGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.dealt || g.status != "playing" {
		return []string{"deal"}
	}
	if !g.drawn {
		return []string{"draw:<indices 0-4>", "show", "comment:pick", "comment:unpick"}
	}
	return []string{"show"}
}

func (g *pokerGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	cmd := strings.ToLower(strings.TrimSpace(uci))
	speakOnly := false

	switch {
	case cmd == "deal":
		if g.dealt && g.status == "playing" {
			g.mu.Unlock()
			return nil, fmt.Errorf("ván đang chơi — hãy Show hoặc Ván mới")
		}
		g.deck = pokerNewDeck()
		g.playerHand = append([]string{}, g.deck[:5]...)
		g.deck = g.deck[5:]
		g.botHand = append([]string{}, g.deck[:5]...)
		g.deck = g.deck[5:]
		g.drawn = false
		g.dealt = true
		g.drawCount = 0
		g.status = "playing"
		g.winner = ""
		g.humanTurn = true
		g.moves++
		g.lastMove = "deal"
		g.roundN++
		g.message = g.commentAfterDeal()

	case strings.HasPrefix(cmd, "comment:"):
		kind := strings.TrimPrefix(cmd, "comment:")
		var vi, en string
		if kind == "pick" {
			vi, en = pokerPick(
				[]string{"Bạn muốn đổi lá này à?", "Lựa chọn thú vị.", "Tôi cũng đang cân nhắc lá đó.", "Có thể đây là quyết định đúng."},
				[]string{"Tossing that one?", "Interesting choice.", "I was eyeing that card too.", "Could be the right call."},
			)
		} else {
			vi, en = pokerPick(
				[]string{"Bạn đổi ý rồi sao?", "Hợp lý.", "Giữ lại cũng được."},
				[]string{"Changed your mind?", "Fair enough.", "Keeping it works too."},
			)
		}
		g.message, _ = viOrEN(vi, en)
		speakOnly = true

	case strings.HasPrefix(cmd, "draw:"):
		if !g.dealt || len(g.playerHand) != 5 {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa chia bài")
		}
		if g.drawn {
			g.mu.Unlock()
			return nil, fmt.Errorf("chỉ được đổi bài 1 lần")
		}
		var idxs []int
		seen := map[int]bool{}
		for _, s := range strings.Split(strings.TrimPrefix(cmd, "draw:"), ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			n, err := strconv.Atoi(s)
			if err != nil || n < 0 || n > 4 || seen[n] {
				g.mu.Unlock()
				return nil, fmt.Errorf("chỉ số lá không hợp lệ (0-4)")
			}
			seen[n] = true
			idxs = append(idxs, n)
		}
		for _, i := range idxs {
			if len(g.deck) == 0 {
				break
			}
			g.playerHand[i] = g.deck[0]
			g.deck = g.deck[1:]
		}
		g.drawn = true
		g.drawCount = len(idxs)
		g.moves++
		g.lastMove = "draw"
		for _, i := range pokerBotDrawIndices(g.botHand) {
			if len(g.deck) == 0 {
				break
			}
			g.botHand[i] = g.deck[0]
			g.deck = g.deck[1:]
		}
		g.message = g.commentAfterDraw()

	case cmd == "show":
		if !g.dealt || len(g.playerHand) != 5 {
			g.mu.Unlock()
			return nil, fmt.Errorf("chưa chia bài")
		}
		g.drawn = true
		g.moves++
		g.lastMove = "show"
		pCat, _, _, _, pTb := pokerEvaluate5(g.playerHand)
		bCat, _, _, _, bTb := pokerEvaluate5(g.botHand)
		cmp := pokerCompareHands(pCat, pTb, bCat, bTb)
		g.refreshEvalLocked()
		result := "draw"
		switch cmp {
		case 1:
			g.status, g.winner, result = "win", "human", "win"
		case -1:
			g.status, g.winner, result = "lose", "bot", "lose"
		default:
			g.status, g.winner, result = "draw", "", "draw"
		}
		g.humanTurn = false
		g.message = g.commentShow(cmp)
		g.history = append(g.history, pokerHistEntry{
			Round: g.roundN, Hand: g.playerName, Result: result,
		})
		if len(g.history) > 8 {
			g.history = g.history[len(g.history)-8:]
		}

	default:
		g.mu.Unlock()
		return nil, fmt.Errorf("lệnh không hợp lệ (deal/draw:i,j/show)")
	}

	resp := g.snapshotLocked()
	msg := g.message
	status := g.status
	winner := g.winner
	last := g.lastMove
	g.mu.Unlock()

	hint := "Poker"
	if last == "deal" {
		hint = "Poker. Đã chia bài."
	} else if last == "draw" {
		hint = "Poker. Đổi bài."
	} else if last == "show" && status == "win" && winner == "human" {
		hint = "Poker. Chúc mừng! Bạn thắng."
	} else if speakOnly {
		hint = "Poker"
	}
	queueGameSpeak("poker", msg, hint)
	return resp, nil
}

// NewPoker builds the simplified 5-card draw poker minigame mod.
func NewPoker() *genericBoardMod {
	return newGenericBoardMod("Poker", "poker",
		"Poker rút bài 5 lá — đấu với bot",
		func() string { return getPoker().summaryText() },
		func() map[string]interface{} { return getPoker().snapshot() },
		func() map[string]interface{} { return getPoker().reset() },
		func(s string) string { return getPoker().setDifficulty(s) },
		func() string { return getPoker().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getPoker().playUCI(u) },
		func() []string { return getPoker().legalUCIs() },
		func() (string, string) {
			vi, en := pokerPick(
				[]string{"Ván Poker mới. Tôi sẽ chia bài cho bạn.", "Chúc bạn may mắn ở bàn Poker.", "Hy vọng hôm nay vận may đứng về phía bạn."},
				[]string{"New Poker round. I'll deal you in.", "Good luck at the table.", "Hope luck is on your side today."},
			)
			say, _ := viOrEN(vi, en)
			return say, "Poker. Ván mới."
		},
	)
}
