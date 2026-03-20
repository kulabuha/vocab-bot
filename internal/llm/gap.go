package llm

import "strings"

const gapBlank = "__________"

// GapSentenceFromExample inserts gapBlank over the collocation in the example sentence.
// Tries exact phrase match first, then case-insensitive first occurrence, then common article variants (a/the).
// Returns "" if no occurrence can be replaced (caller should use a generic gap template).
func GapSentenceFromExample(example, phrase string) string {
	example = strings.TrimSpace(example)
	phrase = strings.TrimSpace(phrase)
	if example == "" || phrase == "" {
		return ""
	}
	for _, p := range phraseVariants(phrase) {
		out := strings.Replace(example, p, gapBlank, 1)
		if strings.Contains(out, gapBlank) {
			return out
		}
		out = replaceFirstInsensitive(example, p, gapBlank)
		if strings.Contains(out, gapBlank) {
			return out
		}
	}
	return ""
}

// phraseVariants returns phrase and small alternations (e.g. "a" vs "the") for LLM/example mismatches.
func phraseVariants(phrase string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(phrase)
	lower := strings.ToLower(phrase)
	if strings.Contains(lower, " a ") {
		add(strings.Replace(phrase, " a ", " the ", 1))
		add(strings.Replace(phrase, " A ", " the ", 1))
	}
	if strings.Contains(lower, " the ") {
		add(strings.Replace(phrase, " the ", " a ", 1))
		add(strings.Replace(phrase, " The ", " a ", 1))
	}
	return out
}

func replaceFirstInsensitive(s, old, new string) string {
	if old == "" {
		return s
	}
	lowerS := strings.ToLower(s)
	lowerOld := strings.ToLower(old)
	i := strings.Index(lowerS, lowerOld)
	if i < 0 {
		return s
	}
	// ASCII-safe slice: phrase is English collocation, byte indices match runes for this use case
	end := i + len(old)
	if end > len(s) {
		return s
	}
	return s[:i] + new + s[end:]
}
