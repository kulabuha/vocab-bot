package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"vocab-bot/internal/bot"
	"vocab-bot/internal/config"
	"vocab-bot/internal/db"
	"vocab-bot/internal/llm"
	"vocab-bot/internal/trainer"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if cfg.TelegramBotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is required")
		os.Exit(1)
	}

	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	repo := db.NewRepoSQLite(sqlDB)
	if err := repo.Init(); err != nil {
		slog.Error("migrations", "err", err)
		os.Exit(1)
	}

	llmClient := llm.NewClient(cfg.LLMAPIBase, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout)
	tr := &trainer.Trainer{Repo: repo, LLM: llmClient}
	handler := &bot.Handler{Trainer: tr}

	telegramBot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		slog.Error("telegram bot", "err", err)
		os.Exit(1)
	}
	slog.Info("bot started", "username", telegramBot.Self.UserName)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := telegramBot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return
		case update := <-updates:
			handler.HandleUpdate(ctx, telegramBot, update)
		}
	}
}
