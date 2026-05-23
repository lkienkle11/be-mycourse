package infra

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"mycourse-io-be/internal/shared/cryptox"
)

// HashPassword hashes a plain-text password using bcrypt (cost 12).
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(b), err
}

// CheckPassword verifies a plain-text password against a bcrypt hash.
func CheckPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// CredentialHMACHex returns a deterministic hex digest for username or password material
// using a secret string as HMAC keying material (system auth).
func CredentialHMACHex(secret, plain string) string {
	return cryptox.CredentialHMACHEXString(secret, plain)
}

const systemAccessTokenTTL = 90 * time.Second

// MintSystemAccessToken issues a short-lived HS256 JWT for system auth.
func MintSystemAccessToken(tokenEnv, usernameSecretHex string) (string, error) {
	return cryptox.MintSystemAccessToken(tokenEnv, usernameSecretHex, systemAccessTokenTTL)
}

// ParseSystemAccessToken validates and returns the subject (username secret hex).
func ParseSystemAccessToken(tokenEnv, tokenStr string) (string, error) {
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
	if !tok.Valid || claims.Subject == "" {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Subject, nil
}
