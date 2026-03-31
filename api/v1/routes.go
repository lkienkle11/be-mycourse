package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/middleware"
)

// RegisterNotAuthenRoutes mounts /api/v1 routes that do not require JWT.
func RegisterNotAuthenRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// RegisterAuthenRoutes mounts /api/v1 routes that require a valid Bearer JWT (user_id in context).
func RegisterAuthenRoutes(rg *gin.RouterGroup) {
	rg.GET("/me/permissions",
		middleware.RequirePermission(constants.CodeProfileRead.CourseRead),
		getMyPermissions,
	)
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
	rb.GET("/users/:userId/direct-permissions", listUserDirectPermissionsInternal)
	rb.POST("/users/:userId/roles", assignUserRoleInternal)
	rb.DELETE("/users/:userId/roles/:roleId", removeUserRoleInternal)
	rb.POST("/users/:userId/direct-permissions", assignUserPermissionInternal)
	rb.DELETE("/users/:userId/direct-permissions/:permissionId", removeUserPermissionInternal)
}
