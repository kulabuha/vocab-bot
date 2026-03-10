package db

import "vocab-bot/internal/domain"

type Repo interface {
	Init() error

	UpsertChatState(chatID int64, mode string) error
	GetChatState(chatID int64) (mode string, refreshCounter int, err error)
	IncRefreshCounter(chatID int64) (newVal int, err error)
	ResetRefreshCounter(chatID int64) error

	InsertCollocations(chatID int64, items []domain.Collocation) (int, error)
	// GetExistingPhrasesBySourceWords returns distinct (phrase, source_word, gap_sentence) for the given source words from any user. Used to reuse phrases when a user adds a word that already exists in the DB (no LLM call).
	GetExistingPhrasesBySourceWords(sourceWords []string) ([]struct{ Phrase, SourceWord, GapSentence string }, error)
	GetCollocationByID(id int64) (*domain.Collocation, error)
	GetNextDueLearning(chatID int64, now int64, limit int) ([]domain.Collocation, error)
	GetAnyLearning(chatID int64, limit int) ([]domain.Collocation, error)
	GetRandomMastered(chatID int64, limit int) ([]domain.Collocation, error)
	UpdateProgressAfterAttempt(collocID int64, newStatus domain.Status, newLevel int, nextDue int64, wrongStreak int) error

	CreateExercise(ex domain.Exercise) (int64, error)
	GetLastExercise(chatID int64) (*domain.Exercise, error)
	LogAttempt(chatID int64, ex *domain.Exercise, answer string, grade domain.GradeResult) error

	Stats(chatID int64) (mastered, learning, newCount int, err error)

	// CleanupUserData removes the user from the bot: deletes attempts, exercises, chat_state.
	// Collocations are not deleted: they are moved to the shared pool (chat_id=0) so other users can reuse them; if the pool already has that phrase, the user's row is deleted. Returns counts of deleted rows and collocationsUnassigned (moved to pool or removed as duplicate).
	CleanupUserData(chatID int64) (attemptsDeleted, exercisesDeleted, collocationsUnassigned int64, err error)
}
