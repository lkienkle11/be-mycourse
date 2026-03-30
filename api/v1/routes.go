package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func RegisterRoutes(rg *gin.RouterGroup) {
	// Parent group uses AuthJWTUnlessPublic: protected routes need Bearer JWT.
	rg.GET("/me/permissions", getMyPermissions)
}

func RegisterInternalRoutes(rg *gin.RouterGroup) {
	rb := rg.Group("/rbac")
	rb.GET("/permissions", listPermissionsInternal)
	rb.POST("/permissions", createPermissionInternal)
	rb.PATCH("/permissions/:id", updatePermissionInternal)
	rb.DELETE("/permissions/:id", deletePermissionInternal)

	rb.GET("/roles", listRolesInternal)
	rb.POST("/roles", createRoleInternal)
	rb.GET("/roles/:id", getRoleInternal)
	rb.PATCH("/roles/:id", updateRoleInternal)
	rb.PUT("/roles/:id/permissions", setRolePermissionsInternal)
	rb.DELETE("/roles/:id", deleteRoleInternal)

	rb.GET("/users/:userId/roles", listUserRolesInternal)
	rb.GET("/users/:userId/permissions", listUserPermissionsInternal)
	rb.POST("/users/:userId/roles", assignUserRoleInternal)
	rb.DELETE("/users/:userId/roles/:roleId", removeUserRoleInternal)
}
