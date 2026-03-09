package llm

import (
	"strings"
	"testing"
)

func TestNormalizeFeedbackNewlines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "literal backslash-n replaced",
			in:   "Line one\nLine two",
			want: "Line one\nLine two",
		},
		{
			name: "literal backslash-n in feedback",
			in:   "You wrote: x\nMistakes:\n1) typo",
			want: "You wrote: x\nMistakes:\n1) typo",
		},
		{
			name: "no literal newline unchanged",
			in:   "Single line",
			want: "Single line",
		},
		{
			name: "multiple literal newlines",
			in:   "A\nB\nC",
			want: "A\nB\nC",
		},
	}
	// Use the two-byte sequence that JSON "\\n" decodes to (backslash + n).
	literalBackslashN := string([]byte{'\\', 'n'})
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			// Simulate what we get from JSON when LLM returns "line1\nline2" with literal \n (i.e. "line1\\nline2" in JSON).
			in := strings.ReplaceAll(tests[i].in, "\n", literalBackslashN)
			got := NormalizeFeedbackNewlines(in)
			if got != tests[i].want {
				t.Errorf("NormalizeFeedbackNewlines(%q) = %q, want %q", in, got, tests[i].want)
			}
		})
	}
}
