package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParseUintPathParam parses a path param (base-10, uint32 range) from gin context.
func ParseUintPathParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	if raw == "" {
		return 0, false
	}

	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}
