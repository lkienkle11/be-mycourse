package utils

import "strconv"

// ClampQueryLimit parses raw (a "limit" query param) as a positive int,
// falling back to def on empty/invalid/non-positive input, and capping at max.
func ClampQueryLimit(raw string, def, max int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
