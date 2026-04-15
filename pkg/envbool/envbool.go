package envbool

import (
	"os"
	"strings"
)

// ParseTrue reports true for common enabled values: true, 1, yes, y, on.
func ParseTrue(v string) bool {
	normalized := strings.TrimSpace(v)
	return strings.EqualFold(normalized, "true") ||
		normalized == "1" ||
		strings.EqualFold(normalized, "yes") ||
		strings.EqualFold(normalized, "y") ||
		strings.EqualFold(normalized, "on")
}

// Enabled reads an environment variable and evaluates it with ParseTrue.
func Enabled(name string) bool {
	return ParseTrue(os.Getenv(name))
}
