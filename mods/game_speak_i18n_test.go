package mods

import "testing"

func TestBuildChessCommentForLangChinese(t *testing.T) {
	chessSpeakLangOverride = "zh-CN"
	t.Cleanup(func() { chessSpeakLangOverride = "" })

	got := buildChessCommentForLang("e2e4", "", "p", "", "playing", "")
	if got != "你走了 兵从 e 2 到 e 4。" {
		t.Fatalf("zh-CN human move: %q", got)
	}
	got = buildChessCommentForLang("", "g8f6", "", "n", "check", "")
	if got != "我走了 马从 g 8 到 f 6。 将军！" {
		t.Fatalf("zh-CN bot check: %q", got)
	}
	got = buildChessCommentForLang("", "h7h8q", "", "p", "checkmate", "black")
	if want := "我走了 兵从 h 7 到 h 8，升变为后。 将死。 我赢了。"; got != want {
		t.Fatalf("zh-CN promo mate: got %q want %q", got, want)
	}
}

func TestBuildChessCommentForLangItalian(t *testing.T) {
	chessSpeakLangOverride = "it"
	t.Cleanup(func() { chessSpeakLangOverride = "" })

	got := buildChessSpokenComment("e2e4", "", "p", "", "playing", "")
	if got != "Hai mosso pedone da e 2 a e 4." {
		t.Fatalf("it human move: %q", got)
	}
}

func TestSpeakNewChinese(t *testing.T) {
	chessSpeakLangOverride = "zh-CN"
	t.Cleanup(func() { chessSpeakLangOverride = "" })

	say, _ := speakNew("New game. You are white. Your move.", "Ván mới. Bạn cầm trắng. Đến lượt bạn.")
	if say != "新对局。你执白棋。该你走了。" {
		t.Fatalf("new game zh-CN: %q", say)
	}
}

func TestViOrENFallsBackEnglishWhenUnknown(t *testing.T) {
	chessSpeakLangOverride = "zh-CN"
	t.Cleanup(func() { chessSpeakLangOverride = "" })

	got, _ := viOrEN("Xin chào đặc biệt", "Hello special untranslated")
	if got != "Hello special untranslated" {
		t.Fatalf("untranslated should stay English, got %q", got)
	}
}

func TestPlaceCommentChineseWin(t *testing.T) {
	chessSpeakLangOverride = "zh-CN"
	t.Cleanup(func() { chessSpeakLangOverride = "" })

	got := buildPlaceSpoken("caro", "h8", "", "win", "human")
	if got != "你下了 h8。 你赢了！" {
		t.Fatalf("caro zh-CN: %q", got)
	}
}
