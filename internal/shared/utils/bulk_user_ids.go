package utils

import "strings"

// PrepareBulkUserIDs trims each ID, skips empty strings, and dedupes while preserving first-seen order.
func PrepareBulkUserIDs(userIDs []string) []string {
	seen := make(map[string]struct{}, len(userIDs))
	out := make([]string, 0, len(userIDs))
	for _, rawID := range userIDs {
		userID := strings.TrimSpace(rawID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		out = append(out, userID)
	}
	return out
}
