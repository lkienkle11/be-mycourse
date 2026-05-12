package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/system/application"
)

// RegisterRoutes wires /api/system/* routes.
// The caller is responsible for mounting the router group at the correct path and
// applying rate-limiting / IP middleware before calling this function.
func RegisterRoutes(g *gin.RouterGroup, svc *application.SystemService) {
	if g == nil || svc == nil {
		return
	}
	h := NewHandler(svc)

	g.POST("/login", h.systemLogin)

	authd := g.Group("")
	authd.Use(middleware.RequireSystemAccessToken(svc))

	authd.POST("/permission-sync-now", h.permissionSyncNow)
	authd.POST("/role-permission-sync-now", h.rolePermissionSyncNow)
	authd.POST("/create-permission-sync-job", h.createPermissionSyncJob)
	authd.POST("/create-role-permission-sync-job", h.createRolePermissionSyncJob)
	authd.POST("/delete-permission-sync-job", h.stopPermissionSyncJob)
	authd.POST("/delete-role-permission-sync-job", h.stopRolePermissionSyncJob)
}
