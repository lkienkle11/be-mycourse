package requestutil

import (
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
