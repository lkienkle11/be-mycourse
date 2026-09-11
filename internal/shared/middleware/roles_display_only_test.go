package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type displayRolePermissionChecker struct{}

func (displayRolePermissionChecker) UserHasAllPermissions(_ string, actions []string) (bool, string, error) {
	return false, actions[0], nil
}

func TestRequirePermissionIgnoresDisplayRoleNames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextUserID, "user-1")
		c.Set(ContextPermissions, map[string]struct{}{})
		c.Set("roles", []string{"admin"})
		c.Next()
	})
	router.GET(
		"/protected",
		RequirePermission(displayRolePermissionChecker{}, "course:update"),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
