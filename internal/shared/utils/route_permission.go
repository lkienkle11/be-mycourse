package utils

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/middleware"
)

// RoutePermission adapts the shared RBAC middleware for delivery route registration.
func RoutePermission(checker middleware.PermissionChecker, actions ...string) gin.HandlerFunc {
	return middleware.RequirePermission(checker, actions...)
}
