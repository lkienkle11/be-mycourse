package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/api/exceptions"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/pkg/token"
)

const (
	ContextUserID   = "user_id"
	ContextJWTRole  = "jwt_role"
	ContextSkipAuth = "skip_auth_public" // set when request matches public API exception
)

// AuthJWT validates Authorization: Bearer <JWT> and sets user_id (and jwt_role) on the context.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireJWT(c) {
			return
		}
		c.Next()
	}
}

// AuthJWTUnlessPublic skips JWT when the request matches the public allowlist (no token, no user_id).
func AuthJWTUnlessPublic(rules []exceptions.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		if exceptions.Match(c.Request.Method, c.Request.URL.Path, rules) {
			c.Set(ContextSkipAuth, true)
			c.Next()
			return
		}
		if !requireJWT(c) {
			return
		}
		c.Next()
	}
}

// requireJWT parses Bearer token and sets context; returns false if response already sent (abort).
func requireJWT(c *gin.Context) bool {
	raw := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing bearer token"})
		return false
	}
	tok := strings.TrimSpace(raw[len(prefix):])
	if tok == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "empty token"})
		return false
	}
	secret := setting.AppSetting.JWTSecret
	if secret == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "jwt not configured"})
		return false
	}
	claims, err := token.Parse(secret, tok)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return false
	}
	if claims.UserID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "token missing user_id"})
		return false
	}
	c.Set(ContextUserID, claims.UserID)
	c.Set(ContextJWTRole, claims.Role)
	return true
}
