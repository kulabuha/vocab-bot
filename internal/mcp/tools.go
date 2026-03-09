package mcp

// AddWordsRequest / AddWordsResult for HTTP API.
type AddWordsRequest struct {
	ChatID int64    `json:"chat_id"`
	Words  []string `json:"words"`
}
type AddWordsResult struct {
	Created int `json:"created"`
}

// NextExerciseRequest / NextExerciseResult for HTTP API.
type NextExerciseRequest struct {
	ChatID int64 `json:"chat_id"`
}
type NextExerciseResult struct {
	ExerciseID int64  `json:"exercise_id"`
	Kind       string `json:"kind"`
	Prompt     string `json:"prompt"`
}

// GradeAnswerRequest / GradeAnswerResult for HTTP API.
type GradeAnswerRequest struct {
	ChatID int64  `json:"chat_id"`
	Answer string `json:"answer"`
}
type GradeAnswerResult struct {
	IsCorrect bool   `json:"is_correct"`
	Score     int    `json:"score"`
	Feedback  string `json:"feedback"`
}

// StatsRequest / StatsResult for HTTP API.
type StatsRequest struct {
	ChatID int64 `json:"chat_id"`
}
type StatsResult struct {
	Mastered int `json:"mastered"`
	Learning int `json:"learning"`
	New      int `json:"new"`
}
