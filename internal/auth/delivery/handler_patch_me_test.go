package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/auth/domain"
)

// patchMeUseCase captures the params PATCH /me hands to the application layer.
type patchMeUseCase struct {
	AuthUseCase
	called bool
	params domain.UpdateMeParams
	err    error
}

func (u *patchMeUseCase) UpdateMe(_ context.Context, _ string, p domain.UpdateMeParams) (*domain.MeProfile, error) {
	u.called = true
	u.params = p
	if u.err != nil {
		return nil, u.err
	}
	slug := ""
	if p.Slug != nil {
		slug = *p.Slug
	}
	return &domain.MeProfile{UserID: "u1", Slug: slug}, nil
}

func performPatchMe(t *testing.T, uc *patchMeUseCase, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/me", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	NewHandler(uc, nil).PatchMe(c)
	return w
}

func TestPatchMeSlugValidAndTrimmed(t *testing.T) {
	uc := &patchMeUseCase{}
	w := performPatchMe(t, uc, `{"slug":"  nguyen-van-a "}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if uc.params.Slug == nil || *uc.params.Slug != "nguyen-van-a" {
		t.Fatalf("slug param = %v, want nguyen-van-a", uc.params.Slug)
	}
	var env struct {
		Data struct {
			Slug string `json:"slug"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Slug != "nguyen-van-a" {
		t.Fatalf("response slug = %q, want nguyen-van-a", env.Data.Slug)
	}
}

func TestPatchMeSlugOmittedLeavesSlugNil(t *testing.T) {
	uc := &patchMeUseCase{}
	w := performPatchMe(t, uc, `{}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if uc.params.Slug != nil {
		t.Fatalf("omitted slug must stay nil, got %q", *uc.params.Slug)
	}
}

func TestPatchMeSlugRejectedInputsNeverReachUseCase(t *testing.T) {
	cases := map[string]string{
		"explicit null":  `{"slug":null}`,
		"empty":          `{"slug":""}`,
		"whitespace":     `{"slug":"   "}`,
		"uppercase":      `{"slug":"Nguyen"}`,
		"underscore":     `{"slug":"a_b"}`,
		"leading hyphen": `{"slug":"-abc"}`,
		"too long":       `{"slug":"` + strings.Repeat("a", 256) + `"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			uc := &patchMeUseCase{}
			w := performPatchMe(t, uc, body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
			}
			if uc.called {
				t.Fatal("use case must not run for an invalid slug")
			}
		})
	}
}

func TestPatchMeSlugReservedFromUseCaseIs400(t *testing.T) {
	uc := &patchMeUseCase{err: domain.ErrUserSlugReserved}
	w := performPatchMe(t, uc, `{"slug":"admin"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
}

func TestToMeResponseIncludesSlug(t *testing.T) {
	resp := toMeResponse(&domain.MeProfile{UserID: "u1", Slug: "nguyen-van-a"})
	if resp.Slug != "nguyen-van-a" {
		t.Fatalf("slug = %q, want nguyen-van-a", resp.Slug)
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"slug":"nguyen-van-a"`) {
		t.Fatalf("JSON %s must contain the slug", data)
	}
}
