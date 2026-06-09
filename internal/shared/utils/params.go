package utils

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/uuidx"
)

// ParseUintPathParam parses a path param (base-10, uint32 range) from gin context.
func ParseUintPathParam(c *gin.Context, name string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(name))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}

// ParseUUIDPathParam parses a UUID path param from gin context.
func ParseUUIDPathParam(c *gin.Context, name string) (string, bool) {
	raw := strings.TrimSpace(c.Param(name))
	if raw == "" || !uuidx.IsValid(raw) {
		return "", false
	}
	return raw, true
}
