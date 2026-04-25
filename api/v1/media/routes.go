package media

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	media := rg.Group("/media/files")
	media.OPTIONS("", optionsMedia)
	media.OPTIONS("/:id", optionsMedia)
	media.OPTIONS("/local/:token", optionsMedia)

	media.GET("", middleware.RequirePermission(constants.AllPermissions.MediaFileRead), listFiles)
	media.POST("", middleware.RequirePermission(constants.AllPermissions.MediaFileCreate), createFile)
	media.GET("/:id", middleware.RequirePermission(constants.AllPermissions.MediaFileRead), getFile)
	media.PUT("/:id", middleware.RequirePermission(constants.AllPermissions.MediaFileUpdate), updateFile)
	media.DELETE("/:id", middleware.RequirePermission(constants.AllPermissions.MediaFileDelete), deleteFile)
	media.GET("/local/:token", middleware.RequirePermission(constants.AllPermissions.MediaFileRead), decodeLocalURL)
}
