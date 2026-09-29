package delivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
)

// validUpdateBasicInfoBodyFields are the required non-slug fields for
// updateBasicInfoRequest — kept minimal but valid so only the slug field
// under test varies.
const validUpdateBasicInfoBodyFields = `"expected_row_version":1,"title":"A valid title","short_description":"A short description over twenty chars","about_course":"delta json content long enough to pass the thirty non-whitespace character minimum","thumbnail_file_id":"11111111-1111-1111-1111-111111111111","course_level_id":"11111111-1111-1111-1111-111111111111","course_topic_id":"11111111-1111-1111-1111-111111111111","tag_ids":["11111111-1111-1111-1111-111111111111"],"skill_ids":["11111111-1111-1111-1111-111111111111"],"outcome_ids":["11111111-1111-1111-1111-111111111111"]`

func newUpdateBasicInfoTestContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/courses/11111111-1111-1111-1111-111111111111/basic-info", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "courseId", Value: "11111111-1111-1111-1111-111111111111"}}
	return c, w
}

// TestUpdateBasicInfoExplicitNullSlugIsRejected guards code-review finding
// #1: updateBasicInfoRequest.Slug used to be a *string, which encoding/json
// sets to nil for BOTH an omitted key and an explicit JSON null — so a
// client sending {"slug": null} silently succeeded and left the slug
// unchanged instead of getting the documented 400. h.svc is left nil: the
// reject path returns before ever touching it, which is itself part of what
// this test proves (a pre-fix bug would have fallen through toward
// h.svc.UpdateBasicInfo instead of stopping here).
func TestUpdateBasicInfoExplicitNullSlugIsRejected(t *testing.T) {
	h := &Handler{}
	c, w := newUpdateBasicInfoTestContext(`{` + validUpdateBasicInfoBodyFields + `,"slug":null}`)

	h.updateBasicInfo(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d, want %d (body: %s)", w.Code, http.StatusBadRequest, w.Body.String())
	}
	var body envelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Code != apperrors.BadRequest {
		t.Fatalf("app code = %d, want %d", body.Code, apperrors.BadRequest)
	}
}

func TestUpdateBasicInfoExplicitEmptySlugIsRejected(t *testing.T) {
	h := &Handler{}
	c, w := newUpdateBasicInfoTestContext(`{` + validUpdateBasicInfoBodyFields + `,"slug":""}`)

	h.updateBasicInfo(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d, want %d (body: %s)", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpdateBasicInfoExplicitWhitespaceSlugIsRejected(t *testing.T) {
	h := &Handler{}
	c, w := newUpdateBasicInfoTestContext(`{` + validUpdateBasicInfoBodyFields + `,"slug":"   "}`)

	h.updateBasicInfo(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d, want %d (body: %s)", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpdateBasicInfoMalformedSlugIsRejected(t *testing.T) {
	h := &Handler{}
	c, w := newUpdateBasicInfoTestContext(`{` + validUpdateBasicInfoBodyFields + `,"slug":"NOT-VALID-Slug"}`)

	h.updateBasicInfo(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d, want %d (body: %s)", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
