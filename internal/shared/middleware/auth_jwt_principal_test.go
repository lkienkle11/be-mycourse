package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	authzdomain "mycourse-io-be/internal/authorization/domain"
	"mycourse-io-be/internal/shared/token"
)

func TestPopulateContextAttachesAuthorizationPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	populateContext(ctx, &token.Claims{
		UserID:      "user-1",
		Permissions: []string{"course:update", "course:update"},
	})

	principal, ok := authzdomain.PrincipalFromContext(ctx.Request.Context())
	if !ok {
		t.Fatal("authorization principal is missing from request context")
	}
	if principal.Type != authzdomain.PrincipalTypeUser || principal.ID != "user-1" {
		t.Fatalf("principal = %#v", principal)
	}
	if _, ok := principal.GlobalPermissions["course:update"]; !ok || len(principal.GlobalPermissions) != 1 {
		t.Fatalf("global permissions = %#v", principal.GlobalPermissions)
	}
}
