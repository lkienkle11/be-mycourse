package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
)

// ActiveUserChecker verifies the authenticated user is not soft-deleted, disabled, or actively banned.
// Implemented by auth application layer and injected at router wiring (same pattern as PermissionChecker).
type ActiveUserChecker interface {
	EnsureActiveUser(ctx context.Context, userID uint) error
}

// RequireActiveUser rejects requests when the JWT user fails accessibility checks in Postgres.
// Must run after AuthJWT.
func RequireActiveUser(checker ActiveUserChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ContextUserID)
		if !ok {
			response.AbortFail(c, http.StatusUnauthorized, apperrors.Unauthorized, "not authenticated", nil)
			return
		}
		userID, _ := v.(uint)
		if err := checker.EnsureActiveUser(c.Request.Context(), userID); err != nil {
			writeActiveUserError(c, err)
			return
		}
		c.Next()
	}
}

func writeActiveUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound):
		response.AbortFail(c, http.StatusNotFound, apperrors.NotFound, "user not found", nil)
	case errors.Is(err, apperrors.ErrUserDisabled):
		response.AbortFail(c, http.StatusForbidden, apperrors.UserDisabled, apperrors.DefaultMessage(apperrors.UserDisabled), nil)
	case errors.Is(err, apperrors.ErrUserBanned):
		response.AbortFail(c, http.StatusForbidden, apperrors.UserBanned, apperrors.DefaultMessage(apperrors.UserBanned), nil)
	default:
		response.AbortFail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	}
}
