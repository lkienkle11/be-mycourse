package utils

import (
	"strings"
)

// ValidateUniqueTrimmedStrings trims each value, rejects blanks with emptyErr,
// rejects duplicates with duplicateErr, and returns a deduped list preserving first-seen order.
func ValidateUniqueTrimmedStrings(values []string, emptyErr, duplicateErr error) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			return nil, emptyErr
		}
		if _, ok := seen[v]; ok {
			return nil, duplicateErr
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out, nil
}
