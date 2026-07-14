// Package i18n provides BCP47 content-locale helpers for taxonomy (and similar entities).
// Write and read paths are intentionally separate — do not use mailtmpl.NormalizeLanguageCode here.
package i18n

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	DefaultLocale = "en"
	// MaxLocaleLen is the DB varchar(16) ceiling for persisted locale keys.
	MaxLocaleLen = 16
)

// CanonicalizeLocale validates and canonicalizes a locale for persistence (write path).
// Rules: language lowercase, region uppercase, script titlecase; region is not stripped.
// Empty or invalid input returns an error (4xx at the HTTP layer) — never remaps to en.
func CanonicalizeLocale(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("locale is required")
	}
	parts := strings.Split(s, "-")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("invalid locale %q", raw)
	}
	lang := parts[0]
	if !isLanguageSubtag(lang) {
		return "", fmt.Errorf("invalid locale language %q", raw)
	}
	out := []string{strings.ToLower(lang)}
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if p == "" {
			return "", fmt.Errorf("invalid locale %q", raw)
		}
		switch {
		case isScriptSubtag(p):
			out = append(out, titlecaseASCII(p))
		case isRegionSubtag(p):
			out = append(out, strings.ToUpper(p))
		default:
			// Allow other extension-like subtags as lowercase alphanumeric (len 1–8).
			if !isExtensionSubtag(p) {
				return "", fmt.Errorf("invalid locale subtag %q in %q", p, raw)
			}
			out = append(out, strings.ToLower(p))
		}
	}
	canon := strings.Join(out, "-")
	if len(canon) > MaxLocaleLen {
		return "", fmt.Errorf("locale %q exceeds max length %d", canon, MaxLocaleLen)
	}
	return canon, nil
}

// NegotiateReadLocale picks the effective read locale.
// Missing/empty/invalid → DefaultLocale (en). Does not 4xx.
func NegotiateReadLocale(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return DefaultLocale
	}
	canon, err := CanonicalizeLocale(s)
	if err != nil {
		return DefaultLocale
	}
	return canon
}

// ResolveText applies exact → base language → default locale → canonical fallback.
// translations maps locale → text; empty values are skipped.
func ResolveText(requested string, translations map[string]string, canonical string) (text string, resolvedLocale string) {
	req := NegotiateReadLocale(requested)
	if translations == nil {
		translations = map[string]string{}
	}
	try := func(loc string) (string, bool) {
		v := strings.TrimSpace(translations[loc])
		if v == "" {
			return "", false
		}
		return v, true
	}
	if v, ok := try(req); ok {
		return v, req
	}
	if base := BaseLanguage(req); base != "" && base != req {
		if v, ok := try(base); ok {
			return v, base
		}
	}
	if v, ok := try(DefaultLocale); ok {
		return v, DefaultLocale
	}
	return strings.TrimSpace(canonical), "canonical"
}

// BaseLanguage returns the primary language subtag (e.g. en-US → en).
func BaseLanguage(locale string) string {
	s := strings.TrimSpace(locale)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '-'); i > 0 {
		return strings.ToLower(s[:i])
	}
	return strings.ToLower(s)
}

// LocaleCandidates returns (exact, base) for translation SQL joins and hydrate paths.
// exact is NegotiateReadLocale(requested); base is BaseLanguage(exact), or DefaultLocale if empty.
func LocaleCandidates(requested string) (exact, base string) {
	exact = NegotiateReadLocale(requested)
	base = BaseLanguage(exact)
	if base == "" {
		base = DefaultLocale
	}
	return exact, base
}

func isLanguageSubtag(s string) bool {
	if len(s) < 2 || len(s) > 8 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func isScriptSubtag(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func isRegionSubtag(s string) bool {
	if len(s) == 2 {
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return false
			}
		}
		return true
	}
	if len(s) == 3 {
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return false
			}
		}
		return true
	}
	return false
}

func isExtensionSubtag(s string) bool {
	if len(s) < 1 || len(s) > 8 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func titlecaseASCII(s string) string {
	s = strings.ToLower(s)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
