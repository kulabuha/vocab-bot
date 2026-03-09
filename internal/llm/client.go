package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"vocab-bot/internal/domain"
)

// maxParseRetries is how many extra attempts we make if the LLM returns invalid JSON.
const maxParseRetries = 2

// extractJSON strips optional markdown code block and returns the first JSON object.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```json or ``` at start
	if strings.HasPrefix(s, "```") {
		s = s[3:]
		if strings.HasPrefix(strings.TrimSpace(s), "json") {
			s = strings.TrimSpace(s)[4:]
		}
		s = strings.TrimSpace(s)
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
		s = strings.TrimSpace(s)
	}
	// Take first { ... last } in case model added text
	if start := strings.Index(s, "{"); start >= 0 {
		if end := strings.LastIndex(s, "}"); end > start {
			s = s[start : end+1]
		}
	}
	return fixTrailingCommas(s)
}

// fixTrailingCommas removes trailing commas before } or ] (invalid in JSON, common in LLM output).
var trailingComma = regexp.MustCompile(`,(\s*[}\]])`)

func fixTrailingCommas(s string) string {
	for trailingComma.MatchString(s) {
		s = trailingComma.ReplaceAllString(s, "$1")
	}
	return s
}

// CollocationItem is one phrase from LLM generation.
type CollocationItem struct {
	Phrase               string `json:"phrase"`
	SourceWord            string `json:"-"` // set from parent
	ExampleProfessional   string `json:"example_professional"`
	ExampleCasual         string `json:"example_casual"`
}

// GenResponse is the JSON structure returned by the collocation generator.
type GenResponse struct {
	Items []struct {
		SourceWord   string             `json:"source_word"`
		Collocations []CollocationItem   `json:"collocations"`
	} `json:"items"`
}

// GradeResponse is the JSON structure returned by the grader.
type GradeResponse struct {
	IsCorrect        bool   `json:"is_correct"`
	Score            int    `json:"score"`
	Feedback         string `json:"feedback"`
	NormalizedAnswer string `json:"normalized_answer"`
	CorrectVariant   string `json:"correct_variant"`
	NativeVariant    string `json:"native_variant"`
}

// OpenAI-compatible request/response.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey, model string, timeout time.Duration) *Client {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) chat(ctx context.Context, system, user string) (string, error) {
	if c.BaseURL == "" {
		return "", fmt.Errorf("llm: LLM_API_BASE required")
	}
	url := c.BaseURL + "/chat/completions"
	if len(c.BaseURL) > 0 && c.BaseURL[len(c.BaseURL)-1] == '/' {
		url = c.BaseURL + "chat/completions"
	}
	body := chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm status %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("llm decode: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: no choices")
	}
	return out.Choices[0].Message.Content, nil
}

// isParseError reports whether the error is from invalid JSON (we can retry).
func isParseError(err error) bool {
	return errors.As(err, new(*json.SyntaxError)) || errors.As(err, new(*json.UnmarshalTypeError))
}

// GenerateCollocations calls the LLM and returns domain collocations (Status=NEW, Level=1).
// Retries up to maxParseRetries times if the response is not valid JSON.
func (c *Client) GenerateCollocations(ctx context.Context, words []string) ([]domain.Collocation, error) {
	if len(words) == 0 {
		return nil, nil
	}
	user := fmt.Sprintf(CollocationGenPrompt, joinWords(words))
	system := "Output ONLY valid JSON. No markdown, no explanation."
	var gen GenResponse
	var lastErr error
	for attempt := 0; attempt <= maxParseRetries; attempt++ {
		content, err := c.chat(ctx, system, user)
		if err != nil {
			return nil, err
		}
		lastErr = json.Unmarshal([]byte(extractJSON(content)), &gen)
		if lastErr == nil {
			break
		}
		if !isParseError(lastErr) || attempt == maxParseRetries {
			return nil, fmt.Errorf("llm parse collocations: %w", lastErr)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("llm parse collocations: %w", lastErr)
	}
	now := time.Now().Unix()
	var out []domain.Collocation
	for _, item := range gen.Items {
		for _, coll := range item.Collocations {
			if coll.Phrase == "" {
				continue
			}
			out = append(out, domain.Collocation{
				Phrase:      coll.Phrase,
				SourceWord:  item.SourceWord,
				Status:      domain.StatusNew,
				Level:       1,
				NextDue:     now,
				WrongStreak: 0,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
	}
	return out, nil
}

func joinWords(words []string) string {
	if len(words) == 0 {
		return ""
	}
	b := []byte(words[0])
	for i := 1; i < len(words); i++ {
		b = append(b, ',', ' ')
		b = append(b, words[i]...)
	}
	return string(b)
}

// GradeRaw runs one grading request and returns the raw JSON string (after extractJSON) and the parsed feedback fields as returned by the LLM (before normalizing newlines).
// Useful for debugging: inspect rawJSON and the parsed strings to see if the model returned literal backslash-n.
func (c *Client) GradeRaw(ctx context.Context, kind, phrase, prompt, userAnswer string) (rawJSON string, feedback, correctVariant, nativeVariant string, err error) {
	user := fmt.Sprintf(GradePrompt, kind, phrase, prompt, userAnswer)
	system := "Return ONLY valid JSON. No markdown."
	content, err := c.chat(ctx, system, user)
	if err != nil {
		return "", "", "", "", err
	}
	rawJSON = extractJSON(content)
	var gr GradeResponse
	if err := json.Unmarshal([]byte(rawJSON), &gr); err != nil {
		return rawJSON, "", "", "", err
	}
	return rawJSON, gr.Feedback, gr.CorrectVariant, gr.NativeVariant, nil
}

// Grade calls the LLM to grade a user answer.
// Retries up to maxParseRetries times if the response is not valid JSON.
func (c *Client) Grade(ctx context.Context, kind, phrase, prompt, userAnswer string) (domain.GradeResult, error) {
	user := fmt.Sprintf(GradePrompt, kind, phrase, prompt, userAnswer)
	system := "Return ONLY valid JSON. No markdown."
	var gr GradeResponse
	var lastErr error
	for attempt := 0; attempt <= maxParseRetries; attempt++ {
		content, err := c.chat(ctx, system, user)
		if err != nil {
			return domain.GradeResult{}, err
		}
		lastErr = json.Unmarshal([]byte(extractJSON(content)), &gr)
		if lastErr == nil {
			return domain.GradeResult{
				IsCorrect:        gr.IsCorrect,
				Score:            gr.Score,
				Feedback:         NormalizeFeedbackNewlines(gr.Feedback),
				NormalizedAnswer: gr.NormalizedAnswer,
				CorrectVariant:   NormalizeFeedbackNewlines(gr.CorrectVariant),
				NativeVariant:    NormalizeFeedbackNewlines(gr.NativeVariant),
			}, nil
		}
		if !isParseError(lastErr) || attempt == maxParseRetries {
			return domain.GradeResult{}, fmt.Errorf("llm parse grade: %w", lastErr)
		}
	}
	return domain.GradeResult{}, fmt.Errorf("llm parse grade: %w", lastErr)
}
