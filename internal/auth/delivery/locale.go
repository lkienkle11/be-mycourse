package delivery

import "strings"

// normalizeRegisterLocale returns "en" or "vi" (default "vi") for email i18n and confirmation links.
func normalizeRegisterLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en":
		return "en"
	default:
		return "vi"
	}
}
