package delivery

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all RBAC routes under the provided router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rb := rg.Group("/rbac")

	rb.GET("/permissions", h.listPermissions)
	rb.POST("/permissions", h.createPermission)
	rb.PATCH("/permissions/:permissionId", h.updatePermission)
	rb.DELETE("/permissions/:permissionId", h.deletePermission)

	rb.GET("/roles", h.listRoles)
	rb.POST("/roles", h.createRole)
	rb.GET("/roles/:id", h.getRole)
	rb.PATCH("/roles/:id", h.updateRole)
	rb.PUT("/roles/:id/permissions", h.setRolePermissions)
	rb.DELETE("/roles/:id", h.deleteRole)

	rb.GET("/users/:userId/roles", h.listUserRoles)
	rb.GET("/users/:userId/permissions", h.listUserPermissions)
	rb.GET("/users/:userId/direct-permissions", h.listUserDirectPermissions)
	rb.POST("/users/:userId/roles", h.assignUserRole)
	rb.DELETE("/users/:userId/roles/:roleId", h.removeUserRole)
	rb.POST("/users/:userId/direct-permissions", h.assignUserPermission)
	rb.DELETE("/users/:userId/direct-permissions/:permissionId", h.removeUserPermission)
}
