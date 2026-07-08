package application

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mycourse-io-be/internal/auth/domain"
)

const (
	xTokenURL   = "https://api.x.com/2/oauth2/token"
	xUsersMeURL = "https://api.x.com/2/users/me?user.fields=profile_image_url,confirmed_email"
)

// XOAuthVerifier exchanges X authorization codes and loads the current user profile.
type XOAuthVerifier struct {
	clientID     string
	clientSecret string
	callbackURL  string
	httpClient   *http.Client
}

func NewXOAuthVerifier(clientID, clientSecret, callbackURL string) *XOAuthVerifier {
	return &XOAuthVerifier{
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (x *XOAuthVerifier) ExchangeCodeAndLoadIdentity(
	ctx context.Context,
	code, codeVerifier, channel string,
) (ExternalIdentityInput, error) {
	accessToken, err := x.exchangeCode(ctx, code, codeVerifier)
	if err != nil {
		return ExternalIdentityInput{}, err
	}
	return x.fetchIdentity(ctx, accessToken, channel)
}

func (x *XOAuthVerifier) exchangeCode(ctx context.Context, code, codeVerifier string) (string, error) {
	extra := url.Values{}
	extra.Set("code_verifier", codeVerifier)
	return exchangeOAuthAuthorizationCode(ctx, x.httpClient, oauthCodeExchangeInput{
		tokenURL:       xTokenURL,
		clientID:       x.clientID,
		clientSecret:   x.clientSecret,
		callbackURL:    x.callbackURL,
		code:           code,
		extraForm:      extra,
		invalidCodeErr: domain.ErrInvalidXCode,
	})
}

func (x *XOAuthVerifier) fetchIdentity(ctx context.Context, accessToken, channel string) (ExternalIdentityInput, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xUsersMeURL, nil)
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidXCode
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidXCode
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return ExternalIdentityInput{}, domain.ErrInvalidXCode
	}

	var parsed xUsersMeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidXCode
	}
	if parsed.Data.ID == "" {
		return ExternalIdentityInput{}, domain.ErrInvalidXCode
	}

	displayName := strings.TrimSpace(parsed.Data.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(parsed.Data.Username)
	}
	email := strings.TrimSpace(parsed.Data.ConfirmedEmail)
	if email == "" {
		email = strings.TrimSpace(parsed.Data.Email)
	}

	return ExternalIdentityInput{
		Provider:      domain.OAuthProviderX,
		ProviderSub:   parsed.Data.ID,
		Email:         email,
		EmailVerified: email != "",
		DisplayName:   displayName,
		PictureURL:    parsed.Data.ProfileImageURL,
		Channel:       channel,
	}, nil
}

type xUsersMeResponse struct {
	Data struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Username        string `json:"username"`
		ProfileImageURL string `json:"profile_image_url"`
		ConfirmedEmail  string `json:"confirmed_email"`
		Email           string `json:"email"`
	} `json:"data"`
}
