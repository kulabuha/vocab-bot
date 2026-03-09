// llm-debug calls the grading LLM once and prints the raw response plus parsed/normalized feedback.
// Use to verify how the model returns newlines (literal \n vs real newlines).
//
// Run from vocab-bot dir with the same env as the bot (e.g. .env with LLM_API_BASE, LLM_MODEL):
//
//	cd vocab-bot && go run ./cmd/llm-debug
//
// Requires LLM to be reachable (Ollama, OpenAI, etc.). Output shows raw JSON, whether
// feedback strings contain literal backslash-n, and the normalized text the user would see.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"vocab-bot/internal/config"
	"vocab-bot/internal/llm"
)

func main() {
	cfg := config.Load()
	if cfg.LLMAPIBase == "" {
		fmt.Fprintln(os.Stderr, "LLM_API_BASE is required (e.g. export LLM_API_BASE=http://localhost:11434/v1)")
		os.Exit(1)
	}

	client := llm.NewClient(cfg.LLMAPIBase, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout)
	ctx := context.Background()

	kind := "MEANING"
	phrase := "mother tongue"
	prompt := "Explain in one short sentence what 'mother tongue' means."
	userAnswer := "This meaning is you native language"

	rawJSON, feedback, correctVariant, nativeVariant, err := client.GradeRaw(ctx, kind, phrase, prompt, userAnswer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GradeRaw error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Raw JSON (snippet around feedback fields) ===")
	fmt.Println(rawJSON)
	fmt.Println()

	fmt.Println("=== Parsed feedback (as returned by JSON decoder) ===")
	fmt.Printf("feedback contains literal \\n (backslash-n): %v\n", containsLiteralBackslashN(feedback))
	fmt.Printf("feedback repr: %q\n", feedback)
	fmt.Println()

	fmt.Println("=== Parsed correct_variant ===")
	fmt.Printf("contains literal \\n: %v\n", containsLiteralBackslashN(correctVariant))
	fmt.Printf("repr: %q\n", correctVariant)
	fmt.Println()

	fmt.Println("=== Parsed native_variant ===")
	fmt.Printf("contains literal \\n: %v\n", containsLiteralBackslashN(nativeVariant))
	fmt.Printf("repr: %q\n", nativeVariant)
	fmt.Println()

	fmt.Println("=== After NormalizeFeedbackNewlines (what user sees) ===")
	fmt.Println("feedback:")
	fmt.Println(llm.NormalizeFeedbackNewlines(feedback))
	fmt.Println("correct_variant:")
	fmt.Println(llm.NormalizeFeedbackNewlines(correctVariant))
	fmt.Println("native_variant:")
	fmt.Println(llm.NormalizeFeedbackNewlines(nativeVariant))
}

func containsLiteralBackslashN(s string) bool {
	return strings.Contains(s, "\\n") || strings.Contains(s, "\x5c\x6e")
}
