package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

// RegisterRoutes mounts auth and/or /me routes onto the provided router groups.
// Either authen or notAuthen may be nil when the caller registers public vs authenticated
// routes in separate mount steps (different middleware chains on each group).
// permChecker is injected so the middleware doesn't import the rbac domain directly.
func RegisterRoutes(
	authen *gin.RouterGroup,
	notAuthen *gin.RouterGroup,
	h *Handler,
	permChecker middleware.PermissionChecker,
) {
	if notAuthen != nil {
		authGroup := notAuthen.Group("/auth")
		authGroup.GET("/csrf", h.CSRFToken)
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/confirm", h.ConfirmEmail)
		authGroup.POST("/refresh", h.RefreshToken)
		authGroup.POST("/logout", h.Logout)
		if setting.OAuthGoogleConfigured() {
			authGroup.POST("/google", h.GoogleLogin)
			authGroup.POST("/google/onetap", h.GoogleOneTap)
			authGroup.POST("/google/mobile", h.GoogleMobile)
		}
		if setting.OAuthXConfigured() {
			authGroup.POST("/x", h.XLogin)
		}
		if setting.OAuthDiscordConfigured() {
			authGroup.POST("/discord", h.DiscordLogin)
		}
	}

	if authen != nil {
		authen.GET("/me", h.GetMe)
		authen.PATCH("/me", h.PatchMe)
		authen.DELETE("/me/hard", h.HardDeleteMe)
		authen.DELETE("/me", h.DeleteMe)
		authen.GET("/me/permissions",
			utils.RoutePermission(permChecker, constants.AllPermissions.UserRead),
			h.GetMyPermissions,
		)
	}
}
