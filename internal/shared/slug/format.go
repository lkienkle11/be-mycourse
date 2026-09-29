package slug

import (
	"regexp"
	"strings"
)

var formatPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// ValidateManualFormat trims raw and checks it against the shared slug format
// (lowercase ASCII a-z0-9-, no leading/trailing hyphen) and maxLen (the
// caller's own column limit — this package has no opinion on it, so every
// caller must pass one; found necessary during code review, which caught
// course's update path skipping length validation entirely, unlike create).
// It does not check uniqueness — callers own that against their own table.
// An empty/whitespace-only input after trimming is reported the same as a
// malformed input: the caller decides whether "no slug supplied" is valid
// for its own flow (course create treats it as "generate one"; course update
// treats it as an error).
func ValidateManualFormat(raw string, maxLen int) (trimmed string, ok bool) {
	trimmed = strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > maxLen || !formatPattern.MatchString(trimmed) {
		return "", false
	}
	return trimmed, true
}
