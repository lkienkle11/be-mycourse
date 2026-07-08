package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type oauthCodeExchangeInput struct {
	tokenURL       string
	clientID       string
	clientSecret   string
	callbackURL    string
	code           string
	extraForm      url.Values
	invalidCodeErr error
}

func exchangeOAuthAuthorizationCode(
	ctx context.Context,
	client *http.Client,
	in oauthCodeExchangeInput,
) (string, error) {
	form := url.Values{}
	for key, values := range in.extraForm {
		for _, value := range values {
			form.Set(key, value)
		}
	}
	form.Set("code", in.code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", in.callbackURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, in.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", in.invalidCodeErr
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	basic := base64.StdEncoding.EncodeToString([]byte(in.clientID + ":" + in.clientSecret))
	req.Header.Set("Authorization", "Basic "+basic)

	resp, err := client.Do(req)
	if err != nil {
		return "", in.invalidCodeErr
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", in.invalidCodeErr
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.AccessToken == "" {
		return "", in.invalidCodeErr
	}
	return parsed.AccessToken, nil
}
