package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mycourse-io-be/internal/auth/domain"
)

const (
	discordTokenURL   = "https://discord.com/api/oauth2/token"
	discordUsersMeURL = "https://discord.com/api/v10/users/@me"
)

// DiscordOAuthVerifier exchanges Discord authorization codes and loads the current user profile.
type DiscordOAuthVerifier struct {
	clientID     string
	clientSecret string
	callbackURL  string
	httpClient   *http.Client
}

func NewDiscordOAuthVerifier(clientID, clientSecret, callbackURL string) *DiscordOAuthVerifier {
	return &DiscordOAuthVerifier{
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (d *DiscordOAuthVerifier) ExchangeCodeAndLoadIdentity(
	ctx context.Context,
	code, channel string,
) (ExternalIdentityInput, error) {
	accessToken, err := d.exchangeCode(ctx, code)
	if err != nil {
		return ExternalIdentityInput{}, err
	}
	return d.fetchIdentity(ctx, accessToken, channel)
}

func (d *DiscordOAuthVerifier) exchangeCode(ctx context.Context, code string) (string, error) {
	return exchangeOAuthAuthorizationCode(ctx, d.httpClient, oauthCodeExchangeInput{
		tokenURL:       discordTokenURL,
		clientID:       d.clientID,
		clientSecret:   d.clientSecret,
		callbackURL:    d.callbackURL,
		code:           code,
		invalidCodeErr: domain.ErrInvalidDiscordCode,
	})
}

func (d *DiscordOAuthVerifier) fetchIdentity(ctx context.Context, accessToken, channel string) (ExternalIdentityInput, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordUsersMeURL, nil)
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidDiscordCode
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidDiscordCode
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return ExternalIdentityInput{}, domain.ErrInvalidDiscordCode
	}

	var parsed discordUsersMeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidDiscordCode
	}
	if parsed.ID == "" {
		return ExternalIdentityInput{}, domain.ErrInvalidDiscordCode
	}

	displayName := strings.TrimSpace(parsed.GlobalName)
	if displayName == "" {
		displayName = strings.TrimSpace(parsed.Username)
	}
	email := strings.TrimSpace(parsed.Email)

	var pictureURL string
	if parsed.Avatar != "" {
		pictureURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", parsed.ID, parsed.Avatar)
	}

	return ExternalIdentityInput{
		Provider:      domain.OAuthProviderDiscord,
		ProviderSub:   parsed.ID,
		Email:         email,
		EmailVerified: parsed.Verified,
		DisplayName:   displayName,
		PictureURL:    pictureURL,
		Channel:       channel,
	}, nil
}

type discordUsersMeResponse struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Email      string `json:"email"`
	Verified   bool   `json:"verified"`
	Avatar     string `json:"avatar"`
}
