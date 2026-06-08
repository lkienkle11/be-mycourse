package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/utils"
)

// RegisterRoutes mounts media file + video endpoints on rg.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	media := rg.Group("/media/files")
	media.OPTIONS("", h.optionsMedia)
	media.OPTIONS("/batch-delete", h.optionsMedia)
	media.OPTIONS("/:id", h.optionsMedia)
	media.OPTIONS("/local/:token", h.optionsMedia)

	media.GET("", utils.RoutePermission(pc, constants.AllPermissions.MediaFileRead), h.listFiles)
	media.GET("/cleanup-metrics", utils.RoutePermission(pc, constants.AllPermissions.MediaFileRead), h.getMediaCleanupMetrics)
	media.POST("/batch-delete", utils.RoutePermission(pc, constants.AllPermissions.MediaFileDelete), h.batchDeleteMediaFiles)
	media.POST("", utils.RoutePermission(pc, constants.AllPermissions.MediaFileCreate), h.createFile)
	media.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.MediaFileRead), h.getFile)
	media.PUT("/:id", utils.RoutePermission(pc, constants.AllPermissions.MediaFileUpdate), h.updateFile)
	media.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.MediaFileDelete), h.deleteFile)
	media.GET("/local/:token", utils.RoutePermission(pc, constants.AllPermissions.MediaFileRead), h.decodeLocalURL)

	videos := rg.Group("/media/videos")
	videos.GET("/:id/status", utils.RoutePermission(pc, constants.AllPermissions.MediaFileRead), h.getVideoStatus)
}

// RegisterWebhookRoutes mounts webhook endpoints on rg (no auth).
func RegisterWebhookRoutes(rg *gin.RouterGroup, h *Handler) {
	webhook := rg.Group("/webhook")
	webhook.POST("/bunny", h.bunnyWebhook)
}
