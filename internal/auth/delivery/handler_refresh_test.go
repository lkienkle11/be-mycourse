package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/auth/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

// refreshUseCase returns a fixed RefreshSession error; the FE depends on how it maps.
type refreshUseCase struct {
	AuthUseCase
	err error
}

func (u *refreshUseCase) RefreshSession(context.Context, string, string) (domain.TokenPairResult, error) {
	return domain.TokenPairResult{}, u.err
}

func performRefresh(t *testing.T, err error, withCredentials bool) (int, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	if withCredentials {
		c.Request.Header.Set("X-Refresh-Token", "refresh")
		c.Request.Header.Set("X-Session-Id", "session")
	}
	NewHandler(&refreshUseCase{err: err}, nil).RefreshToken(c)
	var env struct {
		Code int `json:"code"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &env); e != nil {
		t.Fatal(e)
	}
	return w.Code, env.Code
}

// Contract the FE relies on: it clears its cookies only for 401/403, so a lost
// rotation race (ErrInvalidSession) must be 401, never 500.
func TestRefreshTokenStatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		credential bool
		status     int
		code       int
	}{
		{"lost race / invalid session", domain.ErrInvalidSession, true, http.StatusUnauthorized, apperrors.InvalidSession},
		{"user not found", domain.ErrUserNotFound, true, http.StatusUnauthorized, apperrors.InvalidSession},
		{"refresh token expired", domain.ErrRefreshTokenExpired, true, http.StatusUnauthorized, apperrors.RefreshTokenExpired},
		{"user disabled", domain.ErrUserDisabled, true, http.StatusForbidden, apperrors.UserDisabled},
		{"user banned", domain.ErrUserBanned, true, http.StatusForbidden, apperrors.UserBanned},
		{"internal error", errors.New("db down"), true, http.StatusInternalServerError, apperrors.InternalError},
		{"missing credentials", nil, false, http.StatusBadRequest, apperrors.BadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code := performRefresh(t, tc.err, tc.credential)
			if status != tc.status || code != tc.code {
				t.Fatalf("status/code = %d/%d, want %d/%d", status, code, tc.status, tc.code)
			}
		})
	}
}
