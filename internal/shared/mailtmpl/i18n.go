package mailtmpl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const confirmAccountTemplateName = "confirm_account"

const (
	KeyHTMLLang     = "htmlLang"
	KeySubject      = "subject"
	KeyGreeting     = "greeting"
	KeyBody         = "body"
	KeyCTAButton    = "ctaButton"
	KeyIgnoreNote   = "ignoreNote"
	KeyExpiryNote   = "expiryNote"
	KeyCopyLinkHint = "copyLinkHint"
	KeyCopyright    = "copyright"
)

// NormalizeLanguageCode returns "en" or "vi" (default "vi").
func NormalizeLanguageCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "en":
		return "en"
	default:
		return "vi"
	}
}

// LoadConfirmAccountTranslations reads template/languages/confirm_account/{lang}.js.
func LoadConfirmAccountTranslations(languageCode string) (map[string]string, error) {
	lang := NormalizeLanguageCode(languageCode)
	path := filepath.Join(LanguageTemplateBaseDir, confirmAccountTemplateName, lang+".js")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read confirm_account translations %s: %w", lang, err)
	}
	return parseJSObjectLiteral(raw)
}

// Interpolate replaces {{key}} placeholders with values from vars.
func Interpolate(template string, vars map[string]string) string {
	out := template
	for key, value := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

func parseJSObjectLiteral(raw []byte) (map[string]string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("empty translation file")
	}
	if strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("parse translation object: %w", err)
	}
	return parsed, nil
}
