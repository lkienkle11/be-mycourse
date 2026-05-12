package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"mycourse-io-be/internal/shared/constants"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is embedded in short-lived access JWTs.
// UserID is users.id (numeric PK) — used for RBAC lookups.
// UserCode is users.user_code (UUID) — the external-facing identifier.
type Claims struct {
	UserID      uint     `json:"user_id"`
	UserCode    string   `json:"user_code"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	CreatedAt   int64    `json:"created_at"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// RefreshClaims is embedded in long-lived refresh JWTs (lean — no permissions).
// UUID correlates the JWT with a specific entry in users.refresh_token_session.
type RefreshClaims struct {
	UserID uint   `json:"user_id"`
	UUID   string `json:"uuid"`
	jwt.RegisteredClaims
}

// GenerateAccess signs a short-lived access token carrying user identity and permissions.
func GenerateAccess(secret string, userID uint, userCode, email, displayName string, createdAt time.Time, permissions []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:      userID,
		UserCode:    userCode,
		Email:       email,
		DisplayName: displayName,
		CreatedAt:   createdAt.Unix(),
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// GenerateRefresh signs a long-lived refresh token carrying the user's DB id and a session UUID.
// The UUID must match the refresh_token_uuid stored in users.refresh_token_session for the
// corresponding session string.
func GenerateRefresh(secret string, userID uint, sessionUUID string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		UserID: userID,
		UUID:   sessionUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// GenerateSessionString produces a 128-character hex string derived from the JWT secret
// and 32 cryptographically random bytes (HMAC-SHA512).  The result is unique per call
// and ties the session to the deployment's JWT key.
func GenerateSessionString(jwtSecret string) (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf(constants.MsgGenerateSessionNonce, err)
	}
	mac := hmac.New(sha512.New, []byte(jwtSecret))
	mac.Write(nonce)
	return hex.EncodeToString(mac.Sum(nil)), nil // 64 bytes → 128 hex chars
}

// ParseAccess validates an access token and returns its claims.
func ParseAccess(secret, tokenString string) (*Claims, error) {
	if secret == "" || tokenString == "" {
		return nil, errors.New(constants.MsgJWTMissingSecretOrToken)
	}
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf(constants.MsgJWTUnexpectedSigningMethod, t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New(constants.MsgJWTInvalidToken)
	}
	return claims, nil
}

// ParseRefresh validates a refresh token and returns its claims.
func ParseRefresh(secret, tokenString string) (*RefreshClaims, error) {
	if secret == "" || tokenString == "" {
		return nil, errors.New(constants.MsgJWTMissingSecretOrToken)
	}
	claims := &RefreshClaims{}
	t, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf(constants.MsgJWTUnexpectedSigningMethod, t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New(constants.MsgJWTInvalidToken)
	}
	return claims, nil
}

// ParseRefreshIgnoreExpiry parses a refresh token verifying the signature but ignoring
// the expiry claim.  Used during session rotation: expiry is enforced via the DB record,
// not the JWT, so we still need to extract user_id and uuid from tokens that may have
// already passed their JWT expiry window.
func ParseRefreshIgnoreExpiry(secret, tokenString string) (*RefreshClaims, error) {
	if secret == "" || tokenString == "" {
		return nil, errors.New(constants.MsgJWTMissingSecretOrToken)
	}
	claims := &RefreshClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf(constants.MsgJWTUnexpectedSigningMethod, t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return nil, err
	}
	if claims.UserID == 0 {
		return nil, errors.New(constants.MsgJWTRefreshMissingUserID)
	}
	return claims, nil
}

// Parse is an alias for ParseAccess kept for backward compatibility.
func Parse(secret, tokenString string) (*Claims, error) {
	return ParseAccess(secret, tokenString)
}
