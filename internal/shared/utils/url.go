package utils

import "strings"

func NormalizeBaseURL(value, fallback string) string {
	if v := strings.TrimSpace(value); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(fallback, "/")
}

// JoinURLPathSegments joins base (trimmed, no trailing slash) with path segments trimmed of slashes.
func JoinURLPathSegments(base string, parts ...string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	var out strings.Builder
	out.WriteString(b)
	for _, p := range parts {
		seg := strings.Trim(strings.TrimSpace(p), "/")
		if seg == "" {
			continue
		}
		if out.Len() > 0 {
			out.WriteByte('/')
		}
		out.WriteString(seg)
	}
	return out.String()
}
