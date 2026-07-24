package utils

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CountRunes returns the number of Unicode code points in s (UTF-8 runes).
// Prefer this over len(s) for text length contracts shared with FE.
func CountRunes(s string) int {
	return utf8.RuneCountInString(s)
}

// CountNonWhitespace counts runes that are not Unicode whitespace (after trim).
func CountNonWhitespace(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// CountDeltaNonWhitespace counts visible text in Quill Delta JSON (string inserts only).
func CountDeltaNonWhitespace(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	var payload struct {
		Ops []struct {
			Insert json.RawMessage `json:"insert"`
		} `json:"ops"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return CountNonWhitespace(raw)
	}
	total := 0
	for _, op := range payload.Ops {
		var text string
		if err := json.Unmarshal(op.Insert, &text); err == nil {
			total += CountNonWhitespace(text)
		}
	}
	return total
}
