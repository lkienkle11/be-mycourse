package utils

import "strings"

func NormalizeBaseURL(value, fallback string) string {
	if v := strings.TrimSpace(value); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(fallback, "/")
}
