// Package logger provides logging helpers. TeeErrorHandler duplicates error-level logs to a file for analysis.
package logger

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// TeeErrorHandler wraps inner and appends error-level (and above) records to file as JSON lines.
// Use for ERROR_LOG_PATH: errors are in stderr and in file for later analysis (grep, export, etc.).
type TeeErrorHandler struct {
	inner slog.Handler
	file  io.Writer
	mu    *sync.Mutex
}

// NewTeeErrorHandler returns a handler that forwards all to inner and appends Level >= Error to file.
// file is written with one JSON object per line (time, level, msg, attrs).
func NewTeeErrorHandler(inner slog.Handler, file io.Writer) *TeeErrorHandler {
	return &TeeErrorHandler{inner: inner, file: file, mu: &sync.Mutex{}}
}

func (t *TeeErrorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return t.inner.Enabled(ctx, level)
}

func (t *TeeErrorHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := t.inner.Handle(ctx, r); err != nil {
		return err
	}
	if r.Level < slog.LevelError {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := map[string]any{
		"time":  r.Time.UTC().Format(time.RFC3339),
		"level": r.Level.String(),
		"msg":   r.Message,
	}
	r.Attrs(func(a slog.Attr) bool {
		entry[a.Key] = a.Value.Any()
		return true
	})
	line, err := json.Marshal(entry)
	if err != nil {
		return nil
	}
	line = append(line, '\n')
	_, _ = t.file.Write(line)
	return nil
}

func (t *TeeErrorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TeeErrorHandler{inner: t.inner.WithAttrs(attrs), file: t.file, mu: t.mu} // share mutex
}

func (t *TeeErrorHandler) WithGroup(name string) slog.Handler {
	return &TeeErrorHandler{inner: t.inner.WithGroup(name), file: t.file, mu: t.mu}
}

// OpenErrorLog opens path for append (create if not exists). Caller should close when done.
func OpenErrorLog(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}
