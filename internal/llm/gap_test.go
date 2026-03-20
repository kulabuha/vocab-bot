package llm

import "testing"

func TestGapSentenceFromExample(t *testing.T) {
	tests := []struct {
		name    string
		example string
		phrase  string
		want    string
	}{
		{
			name:    "exact match",
			example: "We need to meet a deadline for the report.",
			phrase:  "meet a deadline",
			want:    "We need to __________ for the report.",
		},
		{
			name:    "article mismatch",
			example: "We need to meet the deadline for the Q3 report.",
			phrase:  "meet a deadline",
			want:    "We need to __________ for the Q3 report.",
		},
		{
			name:    "case insensitive",
			example: "We need to Meet The Deadline today.",
			phrase:  "meet the deadline",
			want:    "We need to __________ today.",
		},
		{
			name:    "already has blank",
			example: "We need to __________ for the report.",
			phrase:  "meet a deadline",
			want:    "We need to __________ for the report.",
		},
		{
			name:    "no match",
			example: "Something unrelated.",
			phrase:  "meet a deadline",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GapSentenceFromExample(tt.example, tt.phrase)
			if got != tt.want {
				t.Errorf("GapSentenceFromExample(%q, %q) = %q, want %q", tt.example, tt.phrase, got, tt.want)
			}
		})
	}
}
