package domain

type ExerciseKind string

const (
	KindMeaning    ExerciseKind = "MEANING"    // Level 1: explain meaning (receptive)
	KindGap        ExerciseKind = "GAP"        // Level 2: fill the gap (controlled production)
	KindFill       ExerciseKind = "FILL"       // Level 3: use in a sentence (free production)
	KindParaphrase ExerciseKind = "PARAPHRASE" // Level 4: paraphrase (transfer)
	KindRefresh    ExerciseKind = "REFRESH"     // Mastered: gap-fill for retention
)

type Exercise struct {
	ID            int64
	ChatID        int64
	CollocationID int64
	Level         int
	Kind          ExerciseKind
	Prompt        string
	AnswerKey     string // optional
	CreatedAt     int64
}

type GradeResult struct {
	IsCorrect        bool
	Score            int
	Feedback         string
	NormalizedAnswer string
	CorrectVariant   string // grammatically correct version
	NativeVariant    string // how a native would say it
}
