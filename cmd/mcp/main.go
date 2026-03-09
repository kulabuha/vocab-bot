package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"vocab-bot/internal/config"
	"vocab-bot/internal/db"
	"vocab-bot/internal/llm"
	"vocab-bot/internal/mcp"
	"vocab-bot/internal/trainer"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /add_words", handleAddWords(tr))
	mux.HandleFunc("POST /next_exercise", handleNextExercise(tr))
	mux.HandleFunc("POST /grade_answer", handleGradeAnswer(tr))
	mux.HandleFunc("GET /stats", handleStats(tr))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: ":" + port, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.Info("mcp server listening", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
	stop()
}

func handleAddWords(tr *trainer.Trainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcp.AddWordsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		n, err := tr.AddWords(r.Context(), req.ChatID, req.Words)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mcp.AddWordsResult{Created: n})
	}
}

func handleNextExercise(tr *trainer.Trainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcp.NextExerciseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ex, err := tr.NextExercise(r.Context(), req.ChatID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res := mcp.NextExerciseResult{}
		if ex != nil {
			res.ExerciseID = ex.ID
			res.Kind = string(ex.Kind)
			res.Prompt = ex.Prompt
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}

func handleGradeAnswer(tr *trainer.Trainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcp.GradeAnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, grade, err := tr.GradeAnswer(r.Context(), req.ChatID, req.Answer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res := mcp.GradeAnswerResult{
			IsCorrect: grade.IsCorrect,
			Score:    grade.Score,
			Feedback: grade.Feedback,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}

func handleStats(tr *trainer.Trainer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chatID := int64(0)
		if q := r.URL.Query().Get("chat_id"); q != "" {
			var n int64
			if _, err := fmt.Sscanf(q, "%d", &n); err == nil {
				chatID = n
			}
		}
		mastered, learning, newCount, err := tr.Stats(r.Context(), chatID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mcp.StatsResult{Mastered: mastered, Learning: learning, New: newCount})
	}
}
