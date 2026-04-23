package requestutil

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/middleware"
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
	raw := c.Param(name)
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}
