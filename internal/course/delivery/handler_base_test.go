package delivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

func newTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/courses", nil)
	return c, w
}

type envelope struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func TestMapCourseErrorSlugConflict(t *testing.T) {
	c, w := newTestGinContext()

	handled := mapCourseError(c, &domain.SlugConflictError{RecommendedSlug: "golang-course-x7k92ab"})
	if !handled {
		t.Fatal("expected mapCourseError to report the error as handled")
	}
	if w.Code != http.StatusConflict {
		t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusConflict)
	}

	var body envelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v (body: %s)", err, w.Body.String())
	}
	if body.Code != apperrors.SlugConflict {
		t.Fatalf("app code = %d, want %d", body.Code, apperrors.SlugConflict)
	}
	if got := body.Data["recommended_slug"]; got != "golang-course-x7k92ab" {
		t.Fatalf("data.recommended_slug = %v, want %q", got, "golang-course-x7k92ab")
	}
}

// TestMapCourseErrorSentinelCasesStillWorkAlongsideSlugConflict is a
// regression guard: the new errors.As(err, &slugConflict) check runs before
// the pre-existing sentinel switch, so every existing case must still route
// correctly (a nil *SlugConflictError check must not accidentally shadow a
// sentinel error).
func TestMapCourseErrorSentinelCasesStillWorkAlongsideSlugConflict(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{"not found", domain.ErrCourseNotFound, http.StatusNotFound, apperrors.NotFound},
		{"owner only", domain.ErrCourseOwnerOnly, http.StatusForbidden, apperrors.Forbidden},
		{"optimistic lock", domain.ErrCourseOptimisticLock, http.StatusConflict, apperrors.Conflict},
		{"invalid slug", domain.ErrCourseInvalidSlug, http.StatusBadRequest, apperrors.BadRequest},
		{"title too short", domain.ErrCourseTitleTooShort, http.StatusBadRequest, apperrors.BadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newTestGinContext()
			if !mapCourseError(c, tt.err) {
				t.Fatal("expected mapCourseError to report the error as handled")
			}
			if w.Code != tt.wantStatus {
				t.Fatalf("HTTP status = %d, want %d", w.Code, tt.wantStatus)
			}
			var body envelope
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("app code = %d, want %d", body.Code, tt.wantCode)
			}
		})
	}
}

func TestMapCourseErrorNilReturnsFalse(t *testing.T) {
	c, _ := newTestGinContext()
	if mapCourseError(c, nil) {
		t.Fatal("expected mapCourseError(nil) to return false")
	}
}
