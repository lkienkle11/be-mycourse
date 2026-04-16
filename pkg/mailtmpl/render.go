// Package mailtmpl compiles and renders HTML email templates.
//
// Template source files live in template/html/email/ (project root).
// This package contains only Go logic; the template directory contains only HTML.
//
// Templates are parsed once at package init from the file system, so the binary
// must be started from the project root directory (the standard run location).
package mailtmpl

import (
	"bytes"
	"html/template"
	"path/filepath"
	"sync"
)

const templateBaseDir = "template/html/email"

var (
	once               sync.Once
	confirmAccountTmpl *template.Template
	initErr            error
)

func loadTemplates() {
	once.Do(func() {
		path := filepath.Join(templateBaseDir, "confirm_account.html")
		confirmAccountTmpl, initErr = template.ParseFiles(path)
	})
}

// ConfirmAccountData holds the dynamic values injected into confirm_account.html.
type ConfirmAccountData struct {
	DisplayName string
	ConfirmURL  string
}

// RenderConfirmAccount renders the account-confirmation email and returns the HTML string.
func RenderConfirmAccount(data ConfirmAccountData) (string, error) {
	loadTemplates()
	if initErr != nil {
		return "", initErr
	}
	var buf bytes.Buffer
	if err := confirmAccountTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
