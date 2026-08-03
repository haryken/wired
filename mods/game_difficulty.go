package mods

import "strings"

// Shared bot difficulty for Chess + Xiangqi web games.
const (
	diffEasy   = "easy"
	diffMedium = "medium"
	diffHard   = "hard"
)

func normalizeGameDifficulty(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "easy", "de", "dễ", "e", "1":
		return diffEasy
	case "hard", "kho", "khó", "h", "3", "difficult":
		return diffHard
	default:
		return diffMedium
	}
}

func difficultyLabelVI(d string) string {
	switch normalizeGameDifficulty(d) {
	case diffEasy:
		return "Dễ"
	case diffHard:
		return "Khó"
	default:
		return "Trung bình"
	}
}
