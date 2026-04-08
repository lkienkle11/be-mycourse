package v1

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/response"
)

// RegisterNotAuthenRoutes mounts /api/v1 routes that do not require JWT.
func RegisterNotAuthenRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", func(c *gin.Context) {
		response.Health(c)
	})

	auth := rg.Group("/auth")
	auth.POST("/register", register)
	auth.POST("/login", login)
	auth.GET("/confirm", confirmEmail)
	auth.POST("/refresh", refreshToken)
}

// RegisterAuthenRoutes mounts /api/v1 routes that require a valid Bearer JWT (user_id in context).
func RegisterAuthenRoutes(rg *gin.RouterGroup) {
	rg.GET("/me", getMe)
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
