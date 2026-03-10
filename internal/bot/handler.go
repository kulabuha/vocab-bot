package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"vocab-bot/internal/llm"
	"vocab-bot/internal/stats"
	"vocab-bot/internal/trainer"
	"vocab-bot/internal/words"
)

var wordSplit = regexp.MustCompile(`[\s,;]+`)

// Input limits to avoid abuse and oversized LLM payloads.
const (
	maxAddInputChars = 2000 // max characters for /add word list
	maxAnswerChars   = 2000 // max characters for training answer
)

type Handler struct {
	Trainer *trainer.Trainer
	Stats   stats.Recorder // optional; use stats.NopRecorder{} when not configured
}

func (h *Handler) HandleUpdate(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return
	}

	switch {
	case text == "/start":
		h.send(bot, chatID, "Hi! I'm a collocation practice bot.\n\n/add — add words (I'll generate collocations)\n/train — start training\n/stats — your stats\n/cleanup — unassign you from all your collocations (words stay in the shared pool); deletes your attempts and state")
		return
	case text == "/add":
		h.handleAdd(ctx, bot, chatID)
		return
	case text == "/train":
		h.handleTrain(ctx, bot, chatID)
		return
	case text == "/stats":
		h.handleStats(ctx, bot, chatID)
		return
	case text == "/cleanup":
		h.handleCleanup(ctx, bot, chatID)
		return
	}


	h.handleMessage(ctx, bot, chatID, text)
}

func (h *Handler) handleAdd(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64) {
	h.Trainer.Repo.UpsertChatState(chatID, "ADDING")
	h.send(bot, chatID, "Send up to %d words (comma or space separated), e.g. deadline, meeting, feedback. Only English-like words are accepted.", words.MaxWordsPerAdd)
}

func (h *Handler) handleTrain(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64) {
	ex, err := h.Trainer.NextExercise(ctx, chatID)
	if err != nil {
		slog.Error("next exercise", "err", err)
		h.send(bot, chatID, "Error: "+err.Error())
		return
	}
	if ex == nil {
		h.send(bot, chatID, "No exercises yet. Add words with /add and send a list (e.g. deadline, task, priority). If you just added words, try /train again.")
		return
	}
	if h.Stats != nil {
		h.Stats.RecordTrain(chatID)
	}
	h.send(bot, chatID, ex.Prompt)
}

func (h *Handler) handleStats(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64) {
	mastered, learning, newCount, err := h.Trainer.Stats(ctx, chatID)
	if err != nil {
		h.send(bot, chatID, "Error: "+err.Error())
		return
	}
	h.send(bot, chatID, "📊 Stats:\n• Mastered: %d\n• In progress: %d\n• New: %d", mastered, learning, newCount)
}

func (h *Handler) handleCleanup(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64) {
	attempts, exercises, unassigned, err := h.Trainer.Repo.CleanupUserData(chatID)
	if err != nil {
		slog.Error("cleanup", "err", err)
		h.send(bot, chatID, "Error: "+err.Error())
		return
	}
	h.send(bot, chatID, "You're unsigned: %d attempt(s) and %d exercise(s) removed; %d collocation(s) moved to the shared pool (words stay for others). You can /add words again to start fresh.", attempts, exercises, unassigned)
}

func (h *Handler) handleMessage(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, text string) {
	mode, _, err := h.Trainer.Repo.GetChatState(chatID)
	if err != nil {
		h.send(bot, chatID, "Error: "+err.Error())
		return
	}

	switch mode {
	case "ADDING":
		if len(text) > maxAddInputChars {
			h.send(bot, chatID, "List too long. Send at most %d characters.", maxAddInputChars)
			return
		}
		rawWords := parseWords(text)
		if len(rawWords) == 0 {
			h.send(bot, chatID, "No words found. Send words separated by comma or space, e.g. deadline, meeting")
			return
		}
		if len(rawWords) > words.MaxWordsPerAdd {
			h.send(bot, chatID, "Send at most %d words per message. You sent %d — please send fewer and try again.", words.MaxWordsPerAdd, len(rawWords))
			return
		}
		valid, invalid := words.Filter(rawWords)
		if len(valid) == 0 {
			h.send(bot, chatID, "No valid English words. Skipped: %s. Use single English words (letters only, 2+ chars), e.g. deadline, meeting.", strings.Join(invalid, ", "))
			return
		}
		if len(invalid) > 0 {
			h.send(bot, chatID, "Skipped (not valid): %s. Adding: %s.", strings.Join(invalid, ", "), strings.Join(valid, ", "))
		}
		n, err := h.Trainer.AddWords(ctx, chatID, valid)
		if err != nil {
			slog.Error("add words", "err", err)
			h.send(bot, chatID, "Error generating collocations: "+err.Error())
			return
		}
		if h.Stats != nil {
			h.Stats.RecordAdd(chatID, len(valid), n)
		}
		if n == 0 {
			h.send(bot, chatID, "Added 0. All phrases already in the bank or try different words. /train")
		} else {
			h.send(bot, chatID, "Added %d collocations (duplicates skipped). You can /train now.", n)
		}
	case "TRAINING":
		if len(text) > maxAnswerChars {
			h.send(bot, chatID, "Answer too long. Send at most %d characters.", maxAnswerChars)
			return
		}
		ex, grade, err := h.Trainer.GradeAnswer(ctx, chatID, text)
		if err != nil {
			h.send(bot, chatID, "Error: "+err.Error())
			return
		}
		if h.Stats != nil {
			h.Stats.RecordAnswer(chatID)
		}
		if grade.IsCorrect {
			msg := "✅ Correct! (%d/100)\n\n%s"
			args := []interface{}{grade.Score, llm.NormalizeFeedbackNewlines(grade.Feedback)}
			if grade.NativeVariant != "" {
				msg += "\n\nNative example: %s"
				args = append(args, llm.NormalizeFeedbackNewlines(grade.NativeVariant))
			}
			if grade.CorrectVariant != "" {
				msg += "\n\nCorrect variant: %s"
				args = append(args, llm.NormalizeFeedbackNewlines(grade.CorrectVariant))
			}
			h.send(bot, chatID, msg, args...)
		} else {
			msg := "❌ Not quite.\n\n" + llm.NormalizeFeedbackNewlines(grade.Feedback)
			correctNorm := llm.NormalizeFeedbackNewlines(grade.CorrectVariant)
			nativeNorm := llm.NormalizeFeedbackNewlines(grade.NativeVariant)
			// Show correct_variant only if it's a real sentence, not just the phrase repeated.
			if correctNorm != "" && correctNorm != ex.AnswerKey {
				msg += "\n\nCorrect variant: " + correctNorm
			}
			if nativeNorm != "" && nativeNorm != ex.AnswerKey {
				msg += "\n\nNative variant: " + nativeNorm
			}
			if (correctNorm == "" || correctNorm == ex.AnswerKey) && (nativeNorm == "" || nativeNorm == ex.AnswerKey) {
				msg += "\n\nCorrect: " + ex.AnswerKey
			}
			h.send(bot, chatID, msg)
		}
		next, _ := h.Trainer.NextExercise(ctx, chatID)
		if next != nil {
			h.send(bot, chatID, "—\n%s", next.Prompt)
		} else {
			h.send(bot, chatID, "No more exercises for now. Send /train again or /add more words.")
		}
	default:
		h.send(bot, chatID, "Use /add or /train")
	}
}

func parseWords(s string) []string {
	parts := wordSplit.Split(s, -1)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (h *Handler) send(bot *tgbotapi.BotAPI, chatID int64, format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	_, _ = bot.Send(tgbotapi.NewMessage(chatID, msg))
}
