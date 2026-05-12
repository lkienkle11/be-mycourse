package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
)

// RegisterRoutes mounts auth + me routes onto the provided router groups.
// permChecker is injected so the middleware doesn't import the rbac domain directly.
func RegisterRoutes(
	authen *gin.RouterGroup,
	notAuthen *gin.RouterGroup,
	h *Handler,
	permChecker middleware.PermissionChecker,
) {
	// Unauthenticated auth routes
	authGroup := notAuthen.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.GET("/confirm", h.ConfirmEmail)
	authGroup.POST("/refresh", h.RefreshToken)

	// Authenticated /me routes
	authen.GET("/me", h.GetMe)
	authen.PATCH("/me", h.PatchMe)
	authen.DELETE("/me", h.DeleteMe)
	authen.GET("/me/permissions",
		middleware.RequirePermission(permChecker, constants.AllPermissions.UserRead),
		h.GetMyPermissions,
	)
}
