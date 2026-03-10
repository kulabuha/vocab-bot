package domain

type Status string

const (
	StatusNew      Status = "NEW"
	StatusLearning Status = "LEARNING"
	StatusMastered Status = "MASTERED"
)

type Collocation struct {
	ID          int64
	Phrase      string
	SourceWord  string
	Status      Status
	Level       int // 1..4 (same for all collocations: 1=MEANING, 2=GAP, 3=FILL, 4=PARAPHRASE; then MASTERED)
	NextDue     int64
	WrongStreak int
	CreatedAt   int64
	UpdatedAt   int64
	// GapSentence is the example sentence with the phrase replaced by __________ (for GAP/REFRESH). Empty if not set.
	GapSentence string
}
