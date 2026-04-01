package token

import (
	"errors"
	"fmt"
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
type RefreshClaims struct {
	UserID uint `json:"user_id"`
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

// GenerateRefresh signs a long-lived refresh token carrying only the user's DB id.
func GenerateRefresh(secret string, userID uint, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseAccess validates an access token and returns its claims.
func ParseAccess(secret, tokenString string) (*Claims, error) {
	if secret == "" || tokenString == "" {
		return nil, errors.New("missing secret or token")
	}
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ParseRefresh validates a refresh token and returns its claims.
func ParseRefresh(secret, tokenString string) (*RefreshClaims, error) {
	if secret == "" || tokenString == "" {
		return nil, errors.New("missing secret or token")
	}
	claims := &RefreshClaims{}
	t, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// Parse is an alias for ParseAccess kept for backward compatibility.
func Parse(secret, tokenString string) (*Claims, error) {
	return ParseAccess(secret, tokenString)
}
