package llm

import (
	"regexp"
	"strings"
)

var whitespace = regexp.MustCompile(`\s+`)

// NormalizeAnswer trims and collapses internal whitespace for grading.
func NormalizeAnswer(s string) string {
	s = strings.TrimSpace(s)
	s = whitespace.ReplaceAllString(s, " ")
	return s
}

// reLiteralNewline matches literal backslash-n (two chars). Go regex: \\ is one backslash, then n.
var reLiteralNewline = regexp.MustCompile(`\\n`)

// reLiteralReturn matches literal backslash-r.
var reLiteralReturn = regexp.MustCompile(`\\r`)

// NormalizeFeedbackNewlines turns literal \n and \r (backslash+n/r from LLM JSON) into real newlines.
// Some LLMs return JSON with "\\n" in strings, which decodes to the two chars backslash+n; this replaces them with actual newlines so the user sees line breaks, not "\n".
// Handles both ASCII backslash and fullwidth reverse solidus (U+FF3C). Call on feedback/variant strings after unmarshalling.
func NormalizeFeedbackNewlines(s string) string {
	// Replace literal backslash-n and backslash-r (raw string `\n` is exactly the two bytes we get when JSON has "\\n").
	for strings.Contains(s, `\n`) {
		s = strings.ReplaceAll(s, `\n`, "\n")
	}
	for strings.Contains(s, `\r`) {
		s = strings.ReplaceAll(s, `\r`, "\r")
	}
	s = reLiteralNewline.ReplaceAllString(s, "\n")
	s = reLiteralReturn.ReplaceAllString(s, "\r")
	return s
}
