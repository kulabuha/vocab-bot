package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"vocab-bot/internal/domain"
)

type RepoSQLite struct {
	db *sql.DB
}

func NewRepoSQLite(db *sql.DB) *RepoSQLite {
	return &RepoSQLite{db: db}
}

func (r *RepoSQLite) Init() error {
	return RunMigrations(r.db)
}

func (r *RepoSQLite) UpsertChatState(chatID int64, mode string) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO chat_state (chat_id, mode, refresh_counter, updated_at)
		VALUES (?, ?, 0, ?)
		ON CONFLICT(chat_id) DO UPDATE SET mode = excluded.mode, updated_at = excluded.updated_at
	`, chatID, mode, now)
	if err != nil {
		return fmt.Errorf("upsert chat_state: %w", err)
	}
	return nil
}

func (r *RepoSQLite) GetChatState(chatID int64) (mode string, refreshCounter int, err error) {
	var rc int
	err = r.db.QueryRowContext(context.Background(),
		`SELECT mode, refresh_counter FROM chat_state WHERE chat_id = ?`, chatID).Scan(&mode, &rc)
	if err == sql.ErrNoRows {
		return "IDLE", 0, nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("get chat_state: %w", err)
	}
	return mode, rc, nil
}

func (r *RepoSQLite) IncRefreshCounter(chatID int64) (newVal int, err error) {
	res, err := r.db.ExecContext(context.Background(),
		`UPDATE chat_state SET refresh_counter = refresh_counter + 1, updated_at = ? WHERE chat_id = ?`,
		time.Now().Unix(), chatID)
	if err != nil {
		return 0, fmt.Errorf("inc refresh_counter: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, fmt.Errorf("chat_state not found for chat_id %d", chatID)
	}
	var rc int
	err = r.db.QueryRowContext(context.Background(), `SELECT refresh_counter FROM chat_state WHERE chat_id = ?`, chatID).Scan(&rc)
	if err != nil {
		return 0, fmt.Errorf("read refresh_counter: %w", err)
	}
	return rc, nil
}

func (r *RepoSQLite) ResetRefreshCounter(chatID int64) error {
	_, err := r.db.ExecContext(context.Background(),
		`UPDATE chat_state SET refresh_counter = 0, updated_at = ? WHERE chat_id = ?`,
		time.Now().Unix(), chatID)
	if err != nil {
		return fmt.Errorf("reset refresh_counter: %w", err)
	}
	return nil
}

func (r *RepoSQLite) InsertCollocations(chatID int64, items []domain.Collocation) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	now := time.Now().Unix()
	for i := range items {
		items[i].CreatedAt = now
		items[i].UpdatedAt = now
		if items[i].NextDue == 0 {
			items[i].NextDue = now
		}
	}
	var inserted int
	for _, c := range items {
		res, err := r.db.ExecContext(context.Background(), `
			INSERT OR IGNORE INTO collocations (chat_id, phrase, source_word, status, level, next_due, wrong_streak, created_at, updated_at, gap_sentence)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, chatID, c.Phrase, c.SourceWord, string(c.Status), c.Level, c.NextDue, c.WrongStreak, c.CreatedAt, c.UpdatedAt, c.GapSentence)
		if err != nil {
			return inserted, fmt.Errorf("insert collocation %q: %w", c.Phrase, err)
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}
	return inserted, nil
}

func (r *RepoSQLite) GetExistingPhrasesBySourceWords(sourceWords []string) ([]struct{ Phrase, SourceWord, GapSentence string }, error) {
	if len(sourceWords) == 0 {
		return nil, nil
	}
	args := make([]any, len(sourceWords))
	for i, w := range sourceWords {
		args[i] = w
	}
	placeholders := strings.Repeat("?,", len(sourceWords))
	placeholders = placeholders[:len(placeholders)-1]
	query := `SELECT phrase, source_word, COALESCE(MAX(CASE WHEN gap_sentence != '' THEN gap_sentence END), '') FROM collocations WHERE source_word IN (` + placeholders + `) GROUP BY phrase, source_word`
	rows, err := r.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("get existing phrases: %w", err)
	}
	defer rows.Close()
	var out []struct{ Phrase, SourceWord, GapSentence string }
	for rows.Next() {
		var p, sw, gs string
		if err := rows.Scan(&p, &sw, &gs); err != nil {
			return nil, fmt.Errorf("scan existing phrase: %w", err)
		}
		out = append(out, struct{ Phrase, SourceWord, GapSentence string }{p, sw, gs})
	}
	return out, rows.Err()
}

func (r *RepoSQLite) GetCollocationByID(id int64) (*domain.Collocation, error) {
	var c domain.Collocation
	var status string
	var gapSentence sql.NullString
	err := r.db.QueryRowContext(context.Background(), `
		SELECT id, phrase, source_word, status, level, next_due, wrong_streak, created_at, updated_at, gap_sentence
		FROM collocations WHERE id = ?
	`, id).Scan(&c.ID, &c.Phrase, &c.SourceWord, &status, &c.Level, &c.NextDue, &c.WrongStreak, &c.CreatedAt, &c.UpdatedAt, &gapSentence)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get collocation: %w", err)
	}
	c.Status = domain.Status(status)
	if gapSentence.Valid {
		c.GapSentence = gapSentence.String
	}
	return &c, nil
}

func (r *RepoSQLite) GetNextDueLearning(chatID int64, now int64, limit int) ([]domain.Collocation, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, phrase, source_word, status, level, next_due, wrong_streak, created_at, updated_at, gap_sentence
		FROM collocations
		WHERE chat_id = ? AND status IN ('NEW','LEARNING') AND next_due <= ?
		ORDER BY next_due ASC, wrong_streak DESC
		LIMIT ?
	`, chatID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("get next due: %w", err)
	}
	defer rows.Close()
	return scanCollocations(rows)
}

func (r *RepoSQLite) GetAnyLearning(chatID int64, limit int) ([]domain.Collocation, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, phrase, source_word, status, level, next_due, wrong_streak, created_at, updated_at, gap_sentence
		FROM collocations
		WHERE chat_id = ? AND status IN ('NEW','LEARNING')
		ORDER BY next_due ASC
		LIMIT ?
	`, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("get any learning: %w", err)
	}
	defer rows.Close()
	return scanCollocations(rows)
}

func (r *RepoSQLite) GetRandomMastered(chatID int64, limit int) ([]domain.Collocation, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, phrase, source_word, status, level, next_due, wrong_streak, created_at, updated_at, gap_sentence
		FROM collocations
		WHERE chat_id = ? AND status = 'MASTERED'
		ORDER BY RANDOM()
		LIMIT ?
	`, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("get random mastered: %w", err)
	}
	defer rows.Close()
	return scanCollocations(rows)
}

func scanCollocations(rows *sql.Rows) ([]domain.Collocation, error) {
	var out []domain.Collocation
	for rows.Next() {
		var c domain.Collocation
		var status string
		var gapSentence sql.NullString
		err := rows.Scan(&c.ID, &c.Phrase, &c.SourceWord, &status, &c.Level, &c.NextDue, &c.WrongStreak, &c.CreatedAt, &c.UpdatedAt, &gapSentence)
		if err != nil {
			return nil, err
		}
		c.Status = domain.Status(status)
		if gapSentence.Valid {
			c.GapSentence = gapSentence.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *RepoSQLite) UpdateProgressAfterAttempt(collocID int64, newStatus domain.Status, newLevel int, nextDue int64, wrongStreak int) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE collocations SET status = ?, level = ?, next_due = ?, wrong_streak = ?, updated_at = ?
		WHERE id = ?
	`, string(newStatus), newLevel, nextDue, wrongStreak, now, collocID)
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	return nil
}

func (r *RepoSQLite) CreateExercise(ex domain.Exercise) (int64, error) {
	ex.CreatedAt = time.Now().Unix()
	res, err := r.db.ExecContext(context.Background(), `
		INSERT INTO exercises (chat_id, collocation_id, level, kind, prompt, answer_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, ex.ChatID, ex.CollocationID, ex.Level, string(ex.Kind), ex.Prompt, ex.AnswerKey, ex.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("create exercise: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

func (r *RepoSQLite) GetLastExercise(chatID int64) (*domain.Exercise, error) {
	var ex domain.Exercise
	var kind string
	err := r.db.QueryRowContext(context.Background(), `
		SELECT id, chat_id, collocation_id, level, kind, prompt, answer_key, created_at
		FROM exercises WHERE chat_id = ? ORDER BY created_at DESC LIMIT 1
	`, chatID).Scan(&ex.ID, &ex.ChatID, &ex.CollocationID, &ex.Level, &kind, &ex.Prompt, &ex.AnswerKey, &ex.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last exercise: %w", err)
	}
	ex.Kind = domain.ExerciseKind(kind)
	return &ex, nil
}

func (r *RepoSQLite) LogAttempt(chatID int64, ex *domain.Exercise, answer string, grade domain.GradeResult) error {
	correct := 0
	if grade.IsCorrect {
		correct = 1
	}
	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO attempts (chat_id, exercise_id, collocation_id, attempt_level, kind, answer, is_correct, score, feedback, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, chatID, ex.ID, ex.CollocationID, ex.Level, string(ex.Kind), answer, correct, grade.Score, grade.Feedback, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("log attempt: %w", err)
	}
	return nil
}

func (r *RepoSQLite) Stats(chatID int64) (mastered, learning, newCount int, err error) {
	err = r.db.QueryRowContext(context.Background(), `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'MASTERED' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'LEARNING' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'NEW' THEN 1 ELSE 0 END), 0)
		FROM collocations WHERE chat_id = ?
	`, chatID).Scan(&mastered, &learning, &newCount)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("stats: %w", err)
	}
	return mastered, learning, newCount, nil
}

// PoolChatID is the chat_id used for the shared phrase pool (unassigned collocations after cleanup).
const PoolChatID int64 = 0

func (r *RepoSQLite) CleanupUserData(chatID int64) (attemptsDeleted, exercisesDeleted, collocationsUnassigned int64, err error) {
	ctx := context.Background()
	resAttempts, err := r.db.ExecContext(ctx, `DELETE FROM attempts WHERE chat_id = ?`, chatID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("delete attempts: %w", err)
	}
	resExercises, err := r.db.ExecContext(ctx, `DELETE FROM exercises WHERE chat_id = ?`, chatID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("delete exercises: %w", err)
	}
	// Unsign user from collocations: move to pool (chat_id=0) so phrases stay for reuse; if pool already has that phrase, delete this row.
	rows, err := r.db.QueryContext(ctx, `SELECT id, phrase FROM collocations WHERE chat_id = ?`, chatID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("list user collocations: %w", err)
	}
	var ids []int64
	var phrases []string
	for rows.Next() {
		var id int64
		var phrase string
		if err := rows.Scan(&id, &phrase); err != nil {
			rows.Close()
			return 0, 0, 0, fmt.Errorf("scan collocation: %w", err)
		}
		ids = append(ids, id)
		phrases = append(phrases, phrase)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("iter collocations: %w", err)
	}
	for i, id := range ids {
		phrase := phrases[i]
		var exists int
		err := r.db.QueryRowContext(ctx, `SELECT 1 FROM collocations WHERE chat_id = ? AND phrase = ? LIMIT 1`, PoolChatID, phrase).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			return 0, 0, 0, fmt.Errorf("check pool for phrase: %w", err)
		}
		if err == sql.ErrNoRows {
			_, err = r.db.ExecContext(ctx, `UPDATE collocations SET chat_id = ? WHERE id = ?`, PoolChatID, id)
		} else {
			_, err = r.db.ExecContext(ctx, `DELETE FROM collocations WHERE id = ?`, id)
		}
		if err != nil {
			return 0, 0, 0, fmt.Errorf("unassign collocation: %w", err)
		}
		collocationsUnassigned++
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM chat_state WHERE chat_id = ?`, chatID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("delete chat_state: %w", err)
	}
	na, _ := resAttempts.RowsAffected()
	ne, _ := resExercises.RowsAffected()
	return na, ne, int64(len(ids)), nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
