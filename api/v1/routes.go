package v1

import (
	"github.com/gin-gonic/gin"

	internalv1 "mycourse-io-be/api/v1/internal"
	taxonomyv1 "mycourse-io-be/api/v1/taxonomy"
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
		middleware.RequirePermission(constants.AllPermissions.UserRead),
		getMyPermissions,
	)
	taxonomyv1.RegisterRoutes(rg)
}

func RegisterInternalRoutes(rg *gin.RouterGroup) {
	internalv1.RegisterRoutes(rg)
}
