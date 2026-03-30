package v1

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/middleware"
	"mycourse-io-be/services"
)

func getMyPermissions(c *gin.Context) {
	uid := c.GetString(middleware.ContextUserID)
	set, err := services.PermissionCodesForUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load permissions"})
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	c.JSON(http.StatusOK, gin.H{"permissions": list})
}
