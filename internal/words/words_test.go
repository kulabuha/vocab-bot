package words

import (
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		in       []string
		wantValid   []string
		wantInvalid []string
	}{
		{
			name:        "empty",
			in:          []string{},
			wantValid:   nil,
			wantInvalid: nil,
		},
		{
			name:        "single valid",
			in:          []string{"deadline"},
			wantValid:   []string{"deadline"},
			wantInvalid: nil,
		},
		{
			name:        "valid lowercase",
			in:          []string{"  meeting  ", "Feedback"},
			wantValid:   []string{"meeting", "feedback"},
			wantInvalid: nil,
		},
		{
			name:        "duplicates dropped",
			in:          []string{"deadline", "deadline", "meeting"},
			wantValid:   []string{"deadline", "meeting"},
			wantInvalid: nil,
		},
		{
			name:        "too short",
			in:          []string{"a", "ab"},
			wantValid:   []string{"ab"},
			wantInvalid: []string{"a"},
		},
		{
			name:        "number rejected",
			in:          []string{"123", "deadline"},
			wantValid:   []string{"deadline"},
			wantInvalid: []string{"123"},
		},
		{
			name:        "numeric string",
			in:          []string{"42", "3.14"},
			wantValid:   nil,
			wantInvalid: []string{"42", "3.14"},
		},
		{
			name:        "non-English symbols",
			in:          []string{"café", "naïve", "word!"},
			wantValid:   nil,
			wantInvalid: []string{"café", "naïve", "word!"},
		},
		{
			name:        "hyphen compound",
			in:          []string{"well-being"},
			wantValid:   []string{"well-being"},
			wantInvalid: nil,
		},
		{
			name:        "apostrophe",
			in:          []string{"don't"},
			wantValid:   []string{"don't"},
			wantInvalid: nil,
		},
		{
			name:        "empty and blanks",
			in:          []string{"", "  ", "ok"},
			wantValid:   []string{"ok"},
			wantInvalid: nil,
		},
		{
			name:        "all invalid",
			in:          []string{"1", "??", "ab"}, // ab is valid
			wantValid:   []string{"ab"},
			wantInvalid: []string{"1", "??"},
		},
		{
			name:        "mixed valid invalid",
			in:          []string{"task", "123", "priority", "a", "task"},
			wantValid:   []string{"task", "priority"},
			wantInvalid: []string{"123", "a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, gotInvalid := Filter(tt.in)
			if !reflect.DeepEqual(gotValid, tt.wantValid) {
				t.Errorf("Filter() valid = %v, want %v", gotValid, tt.wantValid)
			}
			if !reflect.DeepEqual(gotInvalid, tt.wantInvalid) {
				t.Errorf("Filter() invalid = %v, want %v", gotInvalid, tt.wantInvalid)
			}
		})
	}
}

func TestMaxWordsPerAdd(t *testing.T) {
	if MaxWordsPerAdd != 5 {
		t.Errorf("MaxWordsPerAdd = %d, want 5", MaxWordsPerAdd)
	}
}
