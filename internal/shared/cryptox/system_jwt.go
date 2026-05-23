package cryptox

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// MintSystemAccessToken issues a short-lived HS256 JWT; sub holds the username_secret digest (hex).
func MintSystemAccessToken(tokenEnv, usernameSecretHex string, ttl time.Duration) (string, error) {
	if tokenEnv == "" {
		return "", fmt.Errorf("missing token signing secret")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   usernameSecretHex,
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return tok.SignedString(JWTKeyFromEnv(tokenEnv))
}
