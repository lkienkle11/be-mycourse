// Package utils — requestutil.go contains Gin request parsing helpers
// migrated from pkg/requestutil/params.go.
package utils

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/middleware"
)

// CurrentUserID returns the authenticated user's numeric ID from the Gin context (set by AuthJWT).
// Returns 0 when the context key is missing (unauthenticated request).
func CurrentUserID(c *gin.Context) uint {
	v, ok := c.Get(middleware.ContextUserID)
	if !ok {
		return 0
	}
	uid, _ := v.(uint)
	return uid
}

// ParseUintParam parses a uint path parameter by name.
func ParseUintParam(c *gin.Context, name string) (uint, bool) {
	return ParseUintPathParam(c, name)
}

// ParsePermissionIDParam parses and validates a permission_id-style path param (max 10 chars).
func ParsePermissionIDParam(c *gin.Context, name string) (string, bool) {
	s := strings.TrimSpace(c.Param(name))
	if s == "" || len(s) > 10 {
		return "", false
	}
	return s, true
}
