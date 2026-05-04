package requestutil

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/logic/utils"
)

func CurrentUserID(c *gin.Context) uint {
	v, ok := c.Get(middleware.ContextUserID)
	if !ok {
		return 0
	}
	uid, _ := v.(uint)
	return uid
}

func ParseUintParam(c *gin.Context, name string) (uint, bool) {
	return utils.ParseUintPathParam(c, name)
}

// ParsePermissionIDParam parses and validates a permission_id-style path param (max 10 chars).
func ParsePermissionIDParam(c *gin.Context, name string) (string, bool) {
	s := strings.TrimSpace(c.Param(name))
	if s == "" || len(s) > 10 {
		return "", false
	}
	return s, true
}
