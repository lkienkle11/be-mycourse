package infra

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/utils"
)

// ResolveMediaUploadObjectKey selects the object key before upload (explicit key, Bunny empty until GUID, local nano key, R2 eight-digit prefix key).
func ResolveMediaUploadObjectKey(reqObjectKey, filename string, provider string) string {
	if dk := strings.TrimSpace(reqObjectKey); dk != "" {
		return strings.TrimLeft(dk, "/")
	}
	switch provider {
	case constants.FileProviderBunny:
		return ""
	case constants.FileProviderLocal:
		return buildLocalUploadObjectKey("", filename)
	default:
		return BuildObjectStorageKey(filename)
	}
}

func buildLocalUploadObjectKey(defaultKey, filename string) string {
	key := strings.TrimSpace(defaultKey)
	if key != "" {
		return strings.TrimLeft(key, "/")
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = strings.ReplaceAll(strings.TrimSpace(base), " ", "-")
	if base == "" {
		base = "file"
	}
	return fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), base, ext)
}

func sanitizeObjectStorageUploadBase(filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = strings.TrimSpace(base)
	if base == "" {
		base = "file"
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), ".-_")
	if s == "" {
		s = "file"
	}
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s + strings.ToLower(ext)
}

// BuildObjectStorageKey builds default R2 object keys: eight random digits, hyphen, sanitized filename.
func BuildObjectStorageKey(filename string) string {
	return utils.GenerateRandomDigits(8) + "-" + sanitizeObjectStorageUploadBase(filename)
}
