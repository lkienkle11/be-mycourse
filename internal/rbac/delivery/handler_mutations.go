package delivery

import "github.com/gin-gonic/gin"

type rbacResourceKind int

const (
	rbacResourcePermission rbacResourceKind = iota
	rbacResourceRole
)

func (h *Handler) updatePermission(c *gin.Context) { updateRBACResource(c, h, rbacResourcePermission) }
func (h *Handler) updateRole(c *gin.Context)       { updateRBACResource(c, h, rbacResourceRole) }

func (h *Handler) deletePermission(c *gin.Context) { deleteRBACResource(c, h, rbacResourcePermission) }
func (h *Handler) deleteRole(c *gin.Context)       { deleteRBACResource(c, h, rbacResourceRole) }
