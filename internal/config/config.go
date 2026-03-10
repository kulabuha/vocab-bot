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
	// ErrorLogPath: if set, error-level logs are appended to this file (JSON lines) for analysis.
	ErrorLogPath string
	// StatsFilePath: if set, per-user usage stats (add/train/answer counts, last request) are written to this JSON file.
	StatsFilePath string
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
		ErrorLogPath:     os.Getenv("ERROR_LOG_PATH"),
		StatsFilePath:    os.Getenv("STATS_FILE_PATH"),
	}
}
