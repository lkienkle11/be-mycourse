package taxonomy

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	DefaultMaxDescriptionItems = 8
	DefaultMaxDescriptionLen   = 120
)

// ValidateDescriptionParagraphs validates a JSONB array of outcome description strings.
func ValidateDescriptionParagraphs(paragraphs []string, maxItems, maxLen int) error {
	if maxItems <= 0 {
		maxItems = DefaultMaxDescriptionItems
	}
	if maxLen <= 0 {
		maxLen = DefaultMaxDescriptionLen
	}
	if len(paragraphs) > maxItems {
		return errors.New("description exceeds max item count")
	}
	for _, p := range paragraphs {
		s := strings.TrimSpace(p)
		if s == "" {
			return errors.New("description paragraph must be non-empty")
		}
		if utf8.RuneCountInString(s) > maxLen {
			return errors.New("description paragraph exceeds max length")
		}
	}
	return nil
}
