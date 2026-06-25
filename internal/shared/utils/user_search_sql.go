package utils

import "strings"

// UserDisplayNameEmailSearchSQL returns an ILIKE clause for users.display_name and users.email.
// Expects the users table alias to be "u".
func UserDisplayNameEmailSearchSQL(search string) (string, map[string]any) {
	trimmed := strings.TrimSpace(search)
	if trimmed == "" {
		return "", nil
	}
	return " AND (u.display_name ILIKE @search OR u.email ILIKE @search)", map[string]any{
		"search": "%" + trimmed + "%",
	}
}
