package srs

import "time"

// NextDueAfterCorrect returns next_due (unix sec) after a correct answer.
// Level 1→2h, 2→1d, 3→3d, 4→7d (then item is marked MASTERED). See docs/EXERCISE_LEVELS_SPEC.md.
func NextDueAfterCorrect(level int, prevWrongStreak int) int64 {
	now := time.Now()
	switch level {
	case 1:
		return now.Add(2 * time.Hour).Unix()
	case 2:
		return now.Add(24 * time.Hour).Unix()
	case 3:
		return now.Add(72 * time.Hour).Unix() // 3 days
	case 4:
		return now.Add(168 * time.Hour).Unix() // 7 days (then MASTERED)
	default:
		return now.Add(168 * time.Hour).Unix()
	}
}

func NextDueAfterWrong(wrongStreak int) int64 {
	now := time.Now()
	switch {
	case wrongStreak <= 1:
		return now.Add(10 * time.Minute).Unix()
	case wrongStreak == 2:
		return now.Add(30 * time.Minute).Unix()
	default:
		return now.Add(2 * time.Hour).Unix()
	}
}
