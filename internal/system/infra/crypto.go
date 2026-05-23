package infra

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"mycourse-io-be/internal/shared/cryptox"
)

// CredentialHMACHex returns a deterministic hex digest for username or password material
// using app_system_env (or any secret string) as the keying material.
func CredentialHMACHex(secret string, plain string) string {
	return cryptox.CredentialHMACHEXString(secret, plain)
}

const SystemAccessTokenTTL = 90 * time.Second

// MintSystemAccessToken issues a short-lived HS256 JWT; sub holds the username_secret digest (hex).
func MintSystemAccessToken(tokenEnv, usernameSecretHex string) (string, error) {
	return cryptox.MintSystemAccessToken(tokenEnv, usernameSecretHex, SystemAccessTokenTTL)
}

// ParseSystemAccessToken validates signature and expiry and returns RegisteredClaims.Subject (username secret hex).
func ParseSystemAccessToken(tokenEnv, tokenStr string) (usernameSecretHex string, err error) {
	if tokenEnv == "" {
		return "", fmt.Errorf("missing token verification secret")
	}
	key := cryptox.JWTKeyFromEnv(tokenEnv)
	claims := &jwt.RegisteredClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return "", err
	}
	if tok == nil || !tok.Valid {
		return "", fmt.Errorf("invalid token")
	}
	if claims.Subject == "" {
		return "", fmt.Errorf("missing subject")
	}
	return claims.Subject, nil
}
