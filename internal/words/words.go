// Package words provides input validation for /add word lists: max count and English-like filter.
package words

import (
	"regexp"
	"strconv"
	"strings"
)

// MaxWordsPerAdd is the maximum number of words allowed per /add message or add_words request.
const MaxWordsPerAdd = 5

var englishWordRe = regexp.MustCompile(`^[a-zA-Z]+('[a-zA-Z]+)?(-[a-zA-Z]+)*$`)

// Filter returns valid (English-like) and invalid words. Valid: 2+ chars, letters only (hyphen/apostrophe allowed), not a number.
func Filter(words []string) (valid, invalid []string) {
	seen := make(map[string]bool)
	for _, w := range words {
		w = strings.TrimSpace(strings.ToLower(w))
		if w == "" {
			continue
		}
		if len(w) < 2 {
			invalid = append(invalid, w)
			continue
		}
		if _, err := strconv.ParseFloat(w, 64); err == nil {
			invalid = append(invalid, w)
			continue
		}
		if !englishWordRe.MatchString(w) {
			invalid = append(invalid, w)
			continue
		}
		if seen[w] {
			continue
		}
		seen[w] = true
		valid = append(valid, w)
	}
	return valid, invalid
}
