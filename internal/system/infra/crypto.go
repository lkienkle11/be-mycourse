package infra

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CredentialHMACHex returns a deterministic hex digest for username or password material
// using app_system_env (or any secret string) as the keying material.
func CredentialHMACHex(secret string, plain string) string {
	k := sha256.Sum256([]byte(secret))
	mac := hmac.New(sha256.New, k[:])
	_, _ = mac.Write([]byte(plain))
	return hex.EncodeToString(mac.Sum(nil))
}

func jwtKeyFromEnv(tokenEnv string) []byte {
	k := sha256.Sum256([]byte(tokenEnv))
	return k[:]
}

const SystemAccessTokenTTL = 90 * time.Second

// MintSystemAccessToken issues a short-lived HS256 JWT; sub holds the username_secret digest (hex).
func MintSystemAccessToken(tokenEnv, usernameSecretHex string) (string, error) {
	if tokenEnv == "" {
		return "", fmt.Errorf("missing token signing secret")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   usernameSecretHex,
		ExpiresAt: jwt.NewNumericDate(now.Add(SystemAccessTokenTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return tok.SignedString(jwtKeyFromEnv(tokenEnv))
}

// ParseSystemAccessToken validates signature and expiry and returns RegisteredClaims.Subject (username secret hex).
func ParseSystemAccessToken(tokenEnv, tokenStr string) (usernameSecretHex string, err error) {
	if tokenEnv == "" {
		return "", fmt.Errorf("missing token verification secret")
	}
	key := jwtKeyFromEnv(tokenEnv)
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
