// Package brevo wraps Brevo's transactional email API (v3).
package brevo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"mycourse-io-be/pkg/mailtmpl"
	"mycourse-io-be/pkg/setting"
)

const apiURL = "https://api.brevo.com/v3/smtp/email"

type contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

type sendEmailPayload struct {
	Sender      contact   `json:"sender"`
	To          []contact `json:"to"`
	Subject     string    `json:"subject"`
	HTMLContent string    `json:"htmlContent"`
}

// SendConfirmationEmail sends an account confirmation link to the registering user.
func SendConfirmationEmail(toEmail, displayName, confirmURL string) error {
	html, err := mailtmpl.RenderConfirmAccount(mailtmpl.ConfirmAccountData{
		DisplayName: displayName,
		ConfirmURL:  confirmURL,
	})
	if err != nil {
		return fmt.Errorf("brevo: render template: %w", err)
	}

	cfg := setting.BrevoSetting
	payload := sendEmailPayload{
		Sender:      contact{Name: cfg.SenderName, Email: cfg.SenderEmail},
		To:          []contact{{Email: toEmail, Name: displayName}},
		Subject:     "Xác nhận tài khoản MyCourse của bạn",
		HTMLContent: html,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("brevo: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("brevo: build request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("brevo: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("brevo: unexpected status %d", resp.StatusCode)
	}
	return nil
}
