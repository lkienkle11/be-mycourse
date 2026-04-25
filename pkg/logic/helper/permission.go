package helper

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ParsePermissionIDParam parses and validates permission id path param.
func ParsePermissionIDParam(c *gin.Context, name string) (string, bool) {
	s := strings.TrimSpace(c.Param(name))
	if s == "" || len(s) > 10 {
		return "", false
	}
	return s, true
}
