package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBPath           string
	TelegramBotToken string
	LLMAPIBase       string
	LLMAPIKey        string
	LLMModel         string
	LLMTimeout       time.Duration
}

func Load() *Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "file:data.db?_journal_mode=WAL&_busy_timeout=5000"
	}
	timeoutSec := 60
	if s := os.Getenv("LLM_TIMEOUT_SEC"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeoutSec = n
		}
	}
	return &Config{
		DBPath:           dbPath,
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		LLMAPIBase:       os.Getenv("LLM_API_BASE"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModel:         os.Getenv("LLM_MODEL"),
		LLMTimeout:       time.Duration(timeoutSec) * time.Second,
	}
}
