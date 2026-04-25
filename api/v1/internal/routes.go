package internal

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup) {
	rb := rg.Group("/rbac")
	rb.GET("/permissions", listPermissionsInternal)
	rb.POST("/permissions", createPermissionInternal)
	rb.PATCH("/permissions/:permissionId", updatePermissionInternal)
	rb.DELETE("/permissions/:permissionId", deletePermissionInternal)

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
