package mods

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// Trivia tiếng Việt: 20 câu hỏi trắc nghiệm cố định, mỗi ván hỏi 5 câu
// (thứ tự xáo trộn ngẫu nhiên).

const triviaQuestionsPerRound = 5

type triviaQA struct {
	Question string
	Choices  [4]string
	Correct  int
}

var triviaBank = []triviaQA{
	{"Thủ đô của Việt Nam là gì?", [4]string{"Hà Nội", "TP.HCM", "Đà Nẵng", "Huế"}, 0},
	{"Việt Nam hiện có bao nhiêu tỉnh thành?", [4]string{"63", "64", "61", "58"}, 0},
	{"Sông nào chảy qua Việt Nam dài nhất thế giới?", [4]string{"Sông Hồng", "Sông Mê Kông", "Sông Đồng Nai", "Sông Mã"}, 1},
	{"Trái Đất quay quanh gì?", [4]string{"Mặt Trăng", "Mặt Trời", "Sao Hỏa", "Sao Kim"}, 1},
	{"Nước nào có diện tích lớn nhất thế giới?", [4]string{"Trung Quốc", "Canada", "Nga", "Mỹ"}, 2},
	{"1 + 1 bằng mấy?", [4]string{"1", "2", "3", "4"}, 1},
	{"Ai là tác giả của Truyện Kiều?", [4]string{"Nguyễn Du", "Hồ Xuân Hương", "Nguyễn Trãi", "Tố Hữu"}, 0},
	{"Việt Nam giành độc lập năm nào?", [4]string{"1945", "1954", "1975", "1930"}, 0},
	{"Hành tinh nào gần Mặt Trời nhất?", [4]string{"Trái Đất", "Sao Thủy", "Sao Kim", "Sao Hỏa"}, 1},
	{"Nước nào đông dân nhất thế giới hiện nay?", [4]string{"Trung Quốc", "Ấn Độ", "Mỹ", "Indonesia"}, 1},
	{"Con vật nào được gọi là chúa sơn lâm?", [4]string{"Voi", "Hổ", "Sư tử", "Gấu"}, 1},
	{"Vịnh Hạ Long thuộc tỉnh nào?", [4]string{"Quảng Ninh", "Hải Phòng", "Nam Định", "Thanh Hóa"}, 0},
	{"1 giờ có bao nhiêu phút?", [4]string{"30", "45", "60", "90"}, 2},
	{"Đâu là một ngôn ngữ lập trình?", [4]string{"Python", "Excel", "Word", "Chrome"}, 0},
	{"Núi cao nhất Việt Nam là núi nào?", [4]string{"Fansipan", "Bà Đen", "Langbiang", "Yên Tử"}, 0},
	{"Chủ tịch Hồ Chí Minh còn được gọi là gì?", [4]string{"Bác Hồ", "Bác Ba", "Ông Sáu", "Anh Hai"}, 0},
	{"Màu của lá cây thường là gì?", [4]string{"Đỏ", "Xanh lá", "Vàng", "Tím"}, 1},
	{"Đâu là thủ đô nước Pháp?", [4]string{"Paris", "London", "Berlin", "Rome"}, 0},
	{"Nước chiếm khoảng bao nhiêu % bề mặt Trái Đất?", [4]string{"50%", "71%", "90%", "30%"}, 1},
	{"Bộ phận nào dùng để nghe?", [4]string{"Mắt", "Mũi", "Tai", "Miệng"}, 2},
}

type triviaGame struct {
	miniCommon
	order []int // shuffled indices into triviaBank
	qIdx  int
	score int
	total int
}

var (
	triviaMu   sync.Mutex
	triviaInst *triviaGame
)

func getTrivia() *triviaGame {
	triviaMu.Lock()
	defer triviaMu.Unlock()
	if triviaInst == nil {
		triviaInst = newTriviaGame()
	}
	return triviaInst
}

func newTriviaGame() *triviaGame {
	perm := rand.Perm(len(triviaBank))
	n := triviaQuestionsPerRound
	if n > len(perm) {
		n = len(perm)
	}
	g := &triviaGame{
		order: perm[:n],
		total: n,
	}
	g.status = "playing"
	g.humanTurn = true
	g.difficulty = diffMedium
	g.message, _ = speakf("Ván đố vui mới, %d câu hỏi. Trả lời bằng 0-3.", "New trivia round, %d questions. Answer with 0-3.", n)
	return g
}

func (g *triviaGame) currentLocked() *triviaQA {
	if g.qIdx < 0 || g.qIdx >= len(g.order) {
		return nil
	}
	return &triviaBank[g.order[g.qIdx]]
}

func (g *triviaGame) snapshotLocked() map[string]interface{} {
	extra := map[string]interface{}{
		"qIndex": g.qIdx,
		"score":  g.score,
		"total":  g.total,
	}
	if q := g.currentLocked(); q != nil {
		extra["question"] = q.Question
		extra["choices"] = []string{q.Choices[0], q.Choices[1], q.Choices[2], q.Choices[3]}
	} else {
		extra["question"] = ""
		extra["choices"] = []string{}
	}
	return g.baseSnap("trivia", "trivia", extra)
}

func (g *triviaGame) snapshot() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshotLocked()
}

func (g *triviaGame) summaryText() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	q := g.currentLocked()
	qtext := "(hết câu hỏi)"
	if q != nil {
		qtext = q.Question
	}
	return fmt.Sprintf("Đố vui tiếng Việt. Câu %d/%d. Điểm: %d. Câu hiện tại: %s", g.qIdx+1, g.total, g.score, qtext)
}

func (g *triviaGame) reset() map[string]interface{} {
	g.mu.Lock()
	prevDiff := g.difficulty
	g.mu.Unlock()
	ng := newTriviaGame()
	if prevDiff != "" {
		ng.difficulty = prevDiff
	}
	triviaMu.Lock()
	triviaInst = ng
	triviaMu.Unlock()
	return ng.snapshot()
}

func (g *triviaGame) setDifficulty(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.difficulty = normalizeGameDifficulty(level)
	return g.difficulty
}

func (g *triviaGame) getDifficulty() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return normalizeGameDifficulty(g.difficulty)
}

func (g *triviaGame) legalUCIs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.status != "playing" || g.currentLocked() == nil {
		return nil
	}
	return []string{"0", "1", "2", "3"}
}

func (g *triviaGame) playUCI(uci string) (map[string]interface{}, error) {
	g.mu.Lock()
	if g.status != "playing" {
		g.mu.Unlock()
		return nil, fmt.Errorf("game over")
	}
	q := g.currentLocked()
	if q == nil {
		g.mu.Unlock()
		return nil, fmt.Errorf("đã hết câu hỏi")
	}
	ans := strings.ToLower(strings.TrimSpace(uci))
	ans = strings.TrimPrefix(ans, "ans:")
	n, err := strconv.Atoi(strings.TrimSpace(ans))
	if err != nil || n < 0 || n > 3 {
		g.mu.Unlock()
		return nil, fmt.Errorf("đáp án phải là 0, 1, 2 hoặc 3")
	}

	correct := n == q.Correct
	g.moves++
	g.lastMove = strconv.Itoa(n)
	g.history = append(g.history, fmt.Sprintf("Q%d:%d", g.qIdx+1, n))

	var msg string
	if correct {
		g.score++
		msg, _ = viOrEN("Chính xác!", "Correct!")
	} else {
		msg, _ = speakf("Sai rồi! Đáp án đúng là: %s.", "Wrong! The correct answer was: %s.", q.Choices[q.Correct])
	}
	g.qIdx++

	if g.qIdx >= g.total {
		g.status = "win"
		if g.score*2 < g.total {
			g.status = "lose"
		}
		if g.score*2 == g.total {
			g.status = "draw"
		}
		g.winner = ""
		if g.status == "win" {
			g.winner = "human"
		} else if g.status == "lose" {
			g.winner = "bot"
		}
		finalMsg, _ := speakf("Xong! Bạn được %d/%d điểm.", "Done! You scored %d/%d.", g.score, g.total)
		msg = msg + " " + finalMsg
	}
	g.message = msg

	resp := g.snapshotLocked()
	spoken := msg
	g.mu.Unlock()

	queueGameSpeak("trivia", spoken, "Trivia: "+q.Question)
	return resp, nil
}

// NewTrivia builds the Vietnamese trivia minigame mod.
func NewTrivia() *genericBoardMod {
	return newGenericBoardMod("Trivia", "trivia",
		"Đố vui tiếng Việt — 5 câu hỏi trắc nghiệm mỗi ván",
		func() string { return getTrivia().summaryText() },
		func() map[string]interface{} { return getTrivia().snapshot() },
		func() map[string]interface{} { return getTrivia().reset() },
		func(s string) string { return getTrivia().setDifficulty(s) },
		func() string { return getTrivia().getDifficulty() },
		func(u string) (map[string]interface{}, error) { return getTrivia().playUCI(u) },
		func() []string { return getTrivia().legalUCIs() },
		func() (string, string) {
			say, _ := speakf("Ván đố vui mới, %d câu hỏi. Trả lời bằng 0, 1, 2 hoặc 3.",
				"New trivia round, %d questions. Answer with 0, 1, 2 or 3.", triviaQuestionsPerRound)
			return say, "Đố vui. Ván mới."
		},
	)
}
