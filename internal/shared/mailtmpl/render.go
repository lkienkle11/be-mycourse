package mailtmpl

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"sync"
)

var (
	once               sync.Once
	confirmAccountTmpl *template.Template
	initErr            error
)

func loadTemplates() {
	once.Do(func() {
		path := filepath.Join(EmailTemplateBaseDir, "confirm_account.html")
		confirmAccountTmpl, initErr = template.ParseFiles(path)
	})
}

// ConfirmAccountData holds localized values injected into confirm_account.html.
type ConfirmAccountData struct {
	Lang         string
	DisplayName  string
	ConfirmURL   string
	Greeting     string
	Body         string
	CTAButton    string
	IgnoreNote   string
	ExpiryNote   template.HTML
	CopyLinkHint string
	Copyright    string
}

// BuildConfirmAccountData loads translations and prepares template data + subject.
func BuildConfirmAccountData(languageCode, displayName, confirmURL string) (ConfirmAccountData, string, error) {
	texts, err := LoadConfirmAccountTranslations(languageCode)
	if err != nil {
		return ConfirmAccountData{}, "", err
	}
	lang := NormalizeLanguageCode(languageCode)
	subject, ok := texts[KeySubject]
	if !ok || subject == "" {
		return ConfirmAccountData{}, "", fmt.Errorf("missing translation key %q", KeySubject)
	}
	vars := map[string]string{"displayName": displayName}
	data := ConfirmAccountData{
		Lang:         texts[KeyHTMLLang],
		DisplayName:  displayName,
		ConfirmURL:   confirmURL,
		Greeting:     Interpolate(texts[KeyGreeting], vars),
		Body:         texts[KeyBody],
		CTAButton:    texts[KeyCTAButton],
		IgnoreNote:   texts[KeyIgnoreNote],
		ExpiryNote:   template.HTML(texts[KeyExpiryNote]),
		CopyLinkHint: texts[KeyCopyLinkHint],
		Copyright:    texts[KeyCopyright],
	}
	if data.Lang == "" {
		data.Lang = lang
	}
	return data, subject, nil
}

// RenderConfirmAccount renders localized account-confirmation email HTML and subject.
func RenderConfirmAccount(languageCode, displayName, confirmURL string) (html string, subject string, err error) {
	loadTemplates()
	if initErr != nil {
		return "", "", initErr
	}
	data, subject, err := BuildConfirmAccountData(languageCode, displayName, confirmURL)
	if err != nil {
		return "", "", err
	}
	var buf bytes.Buffer
	if err := confirmAccountTmpl.Execute(&buf, data); err != nil {
		return "", "", err
	}
	return buf.String(), subject, nil
}
