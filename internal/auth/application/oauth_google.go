package application

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"

	"mycourse-io-be/internal/auth/domain"
)

const googleRedirectPostMessage = "postmessage"

// GoogleOAuthVerifier validates Google authorization codes and ID tokens.
type GoogleOAuthVerifier struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewGoogleOAuthVerifier(clientID, clientSecret string) *GoogleOAuthVerifier {
	return &GoogleOAuthVerifier{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *GoogleOAuthVerifier) oauthConfig() oauth2.Config {
	return oauth2.Config{
		ClientID:     g.clientID,
		ClientSecret: g.clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  googleRedirectPostMessage,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

func (g *GoogleOAuthVerifier) exchangeContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, g.httpClient)
}

func (g *GoogleOAuthVerifier) ExchangeCodeAndVerify(ctx context.Context, code string) (ExternalIdentityInput, error) {
	cfg := g.oauthConfig()
	tok, err := cfg.Exchange(g.exchangeContext(ctx), code)
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidGoogleCode
	}
	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		return ExternalIdentityInput{}, domain.ErrInvalidGoogleCode
	}
	input, err := g.VerifyIDToken(ctx, rawID)
	if err != nil {
		return ExternalIdentityInput{}, err
	}
	if input.Channel == "" {
		input.Channel = domain.OAuthChannelWebPopupLogin
	}
	return input, nil
}

func (g *GoogleOAuthVerifier) VerifyIDToken(ctx context.Context, rawToken string) (ExternalIdentityInput, error) {
	validator, err := idtoken.NewValidator(ctx, option.WithHTTPClient(g.httpClient))
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidGoogleCode
	}
	payload, err := validator.Validate(ctx, rawToken, g.clientID)
	if err != nil {
		return ExternalIdentityInput{}, domain.ErrInvalidGoogleCode
	}
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	sub := payload.Subject
	if sub == "" {
		return ExternalIdentityInput{}, domain.ErrInvalidGoogleCode
	}
	if !emailVerified {
		return ExternalIdentityInput{}, domain.ErrGoogleEmailNotVerified
	}
	return ExternalIdentityInput{
		Provider:      domain.OAuthProviderGoogle,
		ProviderSub:   sub,
		Email:         email,
		EmailVerified: emailVerified,
		DisplayName:   name,
		PictureURL:    picture,
	}, nil
}
