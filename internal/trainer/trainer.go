// Package trainer implements the collocation training flow:
//
//  1. User sends a list of words (/add → words).
//  2. LLM generates collocations for those words (CollocationGenPrompt); they are stored with level=1, status=NEW/LEARNING.
//  3. Each time we need an exercise, we pick a collocation from the pool (due or any learning).
//  4. Each collocation has its own level (1–3) and status (NEW, LEARNING, MASTERED). Level is per collocation.
//  5. Exercise type is fixed by level (same for every collocation): Level 1 = MEANING, 2 = GAP, 3 = FILL, 4 = PARAPHRASE; MASTERED = REFRESH. See docs/EXERCISE_LEVELS_SPEC.md.
//  6. On correct we advance level (1→2→3→4) or mark MASTERED after level 4; on wrong we keep level and schedule sooner.
//  7. Grading uses GradePrompt in internal/llm/prompts.go; the LLM is given exercise kind and required collocation so it can judge and return feedback + native_variant (always, including on correct).
package trainer

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"vocab-bot/internal/domain"
	"vocab-bot/internal/db"
	"vocab-bot/internal/llm"
	"vocab-bot/internal/srs"
)

type Trainer struct {
	Repo db.Repo
	LLM  *llm.Client
}

func (t *Trainer) AddWords(ctx context.Context, chatID int64, words []string) (int, error) {
	if len(words) == 0 {
		return 0, nil
	}
	// Reuse phrases already in DB (any user) so we don't call the LLM for the same word again.
	existing, err := t.Repo.GetExistingPhrasesBySourceWords(words)
	if err != nil {
		return 0, fmt.Errorf("get existing phrases: %w", err)
	}
	wordsWithExisting := make(map[string]bool)
	for _, e := range existing {
		wordsWithExisting[e.SourceWord] = true
	}
	var wordsToGenerate []string
	for _, w := range words {
		if !wordsWithExisting[w] {
			wordsToGenerate = append(wordsToGenerate, w)
		}
	}
	var total int
	if len(wordsToGenerate) > 0 {
		collocs, err := t.LLM.GenerateCollocations(ctx, wordsToGenerate)
		if err != nil {
			return 0, fmt.Errorf("generate collocations: %w", err)
		}
		if len(collocs) > 0 {
			n, err := t.Repo.InsertCollocations(chatID, collocs)
			if err != nil {
				return 0, fmt.Errorf("insert: %w", err)
			}
			total += n
		}
	}
	if len(existing) > 0 {
		perWord := make(map[string]int)
		templateCollocs := make([]domain.Collocation, 0, len(existing))
		for _, e := range existing {
			if perWord[e.SourceWord] >= domain.MaxCollocationsPerWord {
				continue
			}
			perWord[e.SourceWord]++
			templateCollocs = append(templateCollocs, domain.Collocation{
				Phrase:      e.Phrase,
				SourceWord:  e.SourceWord,
				GapSentence: e.GapSentence,
				Status:      domain.StatusNew,
				Level:       1,
				WrongStreak: 0,
			})
		}
		n, err := t.Repo.InsertCollocations(chatID, templateCollocs)
		if err != nil {
			return 0, fmt.Errorf("insert existing phrases: %w", err)
		}
		total += n
	}
	if err := t.Repo.UpsertChatState(chatID, "IDLE"); err != nil {
		return total, err
	}
	return total, nil
}

func (t *Trainer) NextExercise(ctx context.Context, chatID int64) (*domain.Exercise, error) {
	mode, refreshCounter, err := t.Repo.GetChatState(chatID)
	if err != nil {
		return nil, err
	}
	if mode == "" {
		mode = "IDLE"
	}
	// Ensure chat_state row exists so IncRefreshCounter works; set mode to TRAINING
	if err := t.Repo.UpsertChatState(chatID, "TRAINING"); err != nil {
		return nil, err
	}

	nextCounter := refreshCounter + 1
	now := time.Now().Unix()
	var list []domain.Collocation
	const poolSize = 50
	if nextCounter%5 == 0 {
		list, err = t.Repo.GetRandomMastered(chatID, 5)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			list, err = t.Repo.GetNextDueLearning(chatID, now, poolSize)
		}
	} else {
		list, err = t.Repo.GetNextDueLearning(chatID, now, poolSize)
	}
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		list, err = t.Repo.GetAnyLearning(chatID, poolSize)
		if err != nil || len(list) == 0 {
			return nil, nil
		}
	}
	// Pick randomly from pool so we don't keep showing the same collocation (was list[0] before)
	c := list[rand.Intn(len(list))]
	// Exercise type is fixed by the collocation's current level until it is mastered (see principles: Stage 1→2→3→4).
	kind := exerciseKindForLevel(c.Level, c.Status)
	prompt := buildPrompt(kind, c.Level, c.Phrase, c.SourceWord, c.GapSentence)
	ex := domain.Exercise{
		ChatID:        chatID,
		CollocationID:  c.ID,
		Level:         c.Level,
		Kind:          kind,
		Prompt:        prompt,
		AnswerKey:     c.Phrase,
	}
	id, err := t.Repo.CreateExercise(ex)
	if err != nil {
		return nil, err
	}
	ex.ID = id
	ex.CreatedAt = time.Now().Unix()
	if _, err := t.Repo.IncRefreshCounter(chatID); err != nil {
		return nil, err
	}
	return &ex, nil
}

// exerciseKindForLevel maps each collocation level to exactly one exercise type (same for all collocations).
// Level 1 = MEANING, 2 = GAP, 3 = FILL, 4 = PARAPHRASE; MASTERED = REFRESH. See docs/EXERCISE_LEVELS_SPEC.md.
func exerciseKindForLevel(level int, status domain.Status) domain.ExerciseKind {
	if status == domain.StatusMastered {
		return domain.KindRefresh
	}
	switch level {
	case 1:
		return domain.KindMeaning
	case 2:
		return domain.KindGap
	case 3:
		return domain.KindFill
	case 4:
		return domain.KindParaphrase
	default:
		return domain.KindMeaning
	}
}

// stageLabel returns the level header for UX. Level 1–4 = MEANING, GAP, FILL, PARAPHRASE; REFRESH for mastered.
func stageLabel(kind domain.ExerciseKind, level int) string {
	var name string
	switch kind {
	case domain.KindMeaning:
		name = "Explain meaning"
	case domain.KindGap:
		name = "Fill the gap"
	case domain.KindFill:
		name = "Use in a sentence"
	case domain.KindParaphrase:
		name = "Paraphrase"
	case domain.KindRefresh:
		return "Refresh — Fill the gap"
	default:
		name = "Exercise"
	}
	return "Level " + strconv.Itoa(level) + " — " + name
}

func buildPrompt(kind domain.ExerciseKind, level int, phrase, sourceWord, gapSentence string) string {
	header := stageLabel(kind, level)
	ph := "*" + phrase + "*"
	sw := "*" + sourceWord + "*"
	var body string
	switch kind {
	case domain.KindMeaning:
		body = "What does " + ph + " mean? Explain in your own words (in English)."
	case domain.KindGap:
		// Stored gap_sentence may be the full example if phrase didn't match LLM text exactly; repair here.
		displayGap := llm.GapSentenceFromExample(gapSentence, phrase)
		if displayGap != "" && strings.Contains(displayGap, "__________") {
			if sourceWord != "" {
				body = "Complete the sentence. The missing part is a collocation that includes the word " + sw + " (in English). Reply with the full sentence.\n\n\"" + displayGap + "\""
			} else {
				body = "Complete the sentence. The missing part is a collocation (in English). Reply with the full sentence.\n\n\"" + displayGap + "\""
			}
		} else {
			body = "Complete the sentence using " + ph + " (in English). Reply with the full sentence.\n\n\"She had to __________ before the exam.\""
		}
	case domain.KindFill:
		body = "Use the collocation " + ph + " in one natural sentence (in English)."
	case domain.KindParaphrase:
		body = "Rewrite the following using " + ph + " (in English):\n\n\"He admitted it was his fault.\""
	case domain.KindRefresh:
		displayGap := llm.GapSentenceFromExample(gapSentence, phrase)
		if displayGap != "" && strings.Contains(displayGap, "__________") {
			if sourceWord != "" {
				body = "Complete the sentence. The missing part is a collocation that includes the word " + sw + " (in English). Reply with the full sentence.\n\n\"" + displayGap + "\""
			} else {
				body = "Complete the sentence. The missing part is a collocation (in English). Reply with the full sentence.\n\n\"" + displayGap + "\""
			}
		} else {
			body = "Complete the sentence using " + ph + " (in English). Reply with the full sentence.\n\n\"She had to __________ before the exam.\""
		}
	default:
		body = "What does " + ph + " mean? Explain in your own words (in English)."
	}
	return header + "\n\n" + body
}

func (t *Trainer) GradeAnswer(ctx context.Context, chatID int64, answer string) (*domain.Exercise, *domain.GradeResult, error) {
	ex, err := t.Repo.GetLastExercise(chatID)
	if err != nil || ex == nil {
		return nil, nil, fmt.Errorf("no current exercise")
	}
	colloc, err := t.Repo.GetCollocationByID(ex.CollocationID)
	if err != nil || colloc == nil {
		return nil, nil, fmt.Errorf("collocation not found")
	}
	grade, err := t.LLM.Grade(ctx, string(ex.Kind), colloc.Phrase, ex.Prompt, answer)
	if err != nil {
		return nil, nil, fmt.Errorf("grade: %w", err)
	}
	if err := t.Repo.LogAttempt(chatID, ex, answer, grade); err != nil {
		return nil, &grade, err
	}

	var newStatus domain.Status
	newLevel := colloc.Level
	nextDue := time.Now().Unix()
	wrongStreak := colloc.WrongStreak

	if grade.IsCorrect {
		wrongStreak = 0
		if colloc.Status == domain.StatusMastered {
			// REFRESH correct: keep MASTERED, schedule next review in 7 days
			newStatus = domain.StatusMastered
			newLevel = 4
			nextDue = time.Now().Add(168 * time.Hour).Unix()
		} else {
			nextDue = srs.NextDueAfterCorrect(colloc.Level, colloc.WrongStreak)
			if colloc.Level >= 4 {
				newStatus = domain.StatusMastered
				newLevel = 4
			} else {
				newStatus = domain.StatusLearning
				newLevel = colloc.Level + 1
			}
		}
	} else {
		wrongStreak++
		nextDue = srs.NextDueAfterWrong(wrongStreak)
		if colloc.Status == domain.StatusMastered {
			newStatus = domain.StatusLearning
			newLevel = 2
		} else {
			newStatus = colloc.Status
			newLevel = colloc.Level
		}
	}
	if err := t.Repo.UpdateProgressAfterAttempt(colloc.ID, newStatus, newLevel, nextDue, wrongStreak); err != nil {
		return ex, &grade, err
	}
	return ex, &grade, nil
}

func (t *Trainer) Stats(ctx context.Context, chatID int64) (mastered, learning, newCount int, err error) {
	return t.Repo.Stats(chatID)
}
