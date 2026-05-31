// Package parsebool parses loose boolean strings (env vars, YAML flags, form fields).
package parsebool

import (
	"os"
	"strings"
)

// Loose treats "1", "true", "yes", "y", "on" (case-insensitive) as true; anything else as false.
func Loose(s string) bool {
	normalized := strings.TrimSpace(s)
	return strings.EqualFold(normalized, "true") ||
		normalized == "1" ||
		strings.EqualFold(normalized, "yes") ||
		strings.EqualFold(normalized, "y") ||
		strings.EqualFold(normalized, "on")
}

// EnvEnabled reads an environment variable and evaluates it with Loose.
func EnvEnabled(name string) bool {
	return Loose(os.Getenv(name))
}
