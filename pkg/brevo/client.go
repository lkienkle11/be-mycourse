// Package brevo wraps Brevo's transactional email API (v3).
package brevo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/mailtmpl"
	"mycourse-io-be/pkg/setting"
)

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

func confirmationEmailPayload(toEmail, displayName, html string) sendEmailPayload {
	cfg := setting.BrevoSetting
	return sendEmailPayload{
		Sender:      contact{Name: cfg.SenderName, Email: cfg.SenderEmail},
		To:          []contact{{Email: toEmail, Name: displayName}},
		Subject:     "Xác nhận tài khoản MyCourse của bạn",
		HTMLContent: html,
	}
}

func postBrevoSMTP(payload sendEmailPayload) error {
	cfg := setting.BrevoSetting
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(constants.MsgBrevoMarshalPayload, err)
	}
	req, err := http.NewRequest(http.MethodPost, constants.BrevoSMTPAPIURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf(constants.MsgBrevoBuildRequest, err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(constants.MsgBrevoSendRequest, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf(constants.MsgBrevoUnexpectedStatus, resp.StatusCode)
	}
	return nil
}

// SendConfirmationEmail sends an account confirmation link to the registering user.
func SendConfirmationEmail(toEmail, displayName, confirmURL string) error {
	html, err := mailtmpl.RenderConfirmAccount(mailtmpl.ConfirmAccountData{
		DisplayName: displayName,
		ConfirmURL:  confirmURL,
	})
	if err != nil {
		return fmt.Errorf(constants.MsgBrevoRenderTemplate, err)
	}
	return postBrevoSMTP(confirmationEmailPayload(toEmail, displayName, html))
}
