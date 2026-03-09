package db

import "vocab-bot/internal/domain"

type Repo interface {
	Init() error

	UpsertChatState(chatID int64, mode string) error
	GetChatState(chatID int64) (mode string, refreshCounter int, err error)
	IncRefreshCounter(chatID int64) (newVal int, err error)
	ResetRefreshCounter(chatID int64) error

	InsertCollocations(items []domain.Collocation) (int, error)
	GetCollocationByID(id int64) (*domain.Collocation, error)
	GetNextDueLearning(now int64, limit int) ([]domain.Collocation, error)
	GetAnyLearning(limit int) ([]domain.Collocation, error)
	GetRandomMastered(limit int) ([]domain.Collocation, error)
	UpdateProgressAfterAttempt(collocID int64, newStatus domain.Status, newLevel int, nextDue int64, wrongStreak int) error

	CreateExercise(ex domain.Exercise) (int64, error)
	GetLastExercise(chatID int64) (*domain.Exercise, error)
	LogAttempt(chatID int64, ex *domain.Exercise, answer string, grade domain.GradeResult) error

	Stats(chatID int64) (mastered, learning, newCount int, err error)

	// CleanupUserData deletes all data for the given chat: attempts, exercises, chat_state,
	// and collocations that are only referenced by this chat's exercises. Returns counts of deleted rows.
	CleanupUserData(chatID int64) (attemptsDeleted, exercisesDeleted, collocationsDeleted int64, err error)
}
