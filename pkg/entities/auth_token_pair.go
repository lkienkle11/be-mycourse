package entities

import "time"

// TokenPairResult carries access/refresh tokens and session metadata after login, confirm, or refresh.
type TokenPairResult struct {
	AccessToken  string
	RefreshToken string
	// SessionStr is the 128-char hex string that identifies this session.
	SessionStr string
	// RefreshTTL is the lifetime of the newly issued refresh token.
	RefreshTTL time.Duration
}
