package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
)

// RegisterRoutes mounts media file + video endpoints on rg.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	rp := func(actions ...string) gin.HandlerFunc { return middleware.RequirePermission(pc, actions...) }

	media := rg.Group("/media/files")
	media.OPTIONS("", h.optionsMedia)
	media.OPTIONS("/batch-delete", h.optionsMedia)
	media.OPTIONS("/:id", h.optionsMedia)
	media.OPTIONS("/local/:token", h.optionsMedia)

	media.GET("", rp(constants.AllPermissions.MediaFileRead), h.listFiles)
	media.GET("/cleanup-metrics", rp(constants.AllPermissions.MediaFileRead), h.getMediaCleanupMetrics)
	media.POST("/batch-delete", rp(constants.AllPermissions.MediaFileDelete), h.batchDeleteMediaFiles)
	media.POST("", rp(constants.AllPermissions.MediaFileCreate), h.createFile)
	media.GET("/:id", rp(constants.AllPermissions.MediaFileRead), h.getFile)
	media.PUT("/:id", rp(constants.AllPermissions.MediaFileUpdate), h.updateFile)
	media.DELETE("/:id", rp(constants.AllPermissions.MediaFileDelete), h.deleteFile)
	media.GET("/local/:token", rp(constants.AllPermissions.MediaFileRead), h.decodeLocalURL)

	videos := rg.Group("/media/videos")
	videos.GET("/:id/status", rp(constants.AllPermissions.MediaFileRead), h.getVideoStatus)
}

// RegisterWebhookRoutes mounts webhook endpoints on rg (no auth).
func RegisterWebhookRoutes(rg *gin.RouterGroup, h *Handler) {
	webhook := rg.Group("/webhook")
	webhook.POST("/bunny", h.bunnyWebhook)
}
