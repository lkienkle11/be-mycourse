package tests

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/mapping"
	pkgmedia "mycourse-io-be/pkg/media"
)

func TestBindCreateFileMultipart_ignoresClientKindAndMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/media/files", nil)
	c.Request.PostForm = url.Values{
		"kind":       []string{"VIDEO"},
		"metadata":   []string{`{"page_count":999,"has_password":true}`},
		"object_key": []string{"abc"},
	}

	req, err := mapping.BindCreateFileMultipart(c)
	if err != nil {
		t.Fatalf("bind should accept metadata payload for backward-compat: %v", err)
	}
	if req.Kind != "" {
		t.Fatalf("kind must be server-owned, got %q", req.Kind)
	}
	if req.Metadata != nil {
		t.Fatalf("metadata must be ignored from client input")
	}
}

func TestBuildTypedMetadata_usesDefaultValuesWhenUnavailable(t *testing.T) {
	meta := pkgmedia.BuildTypedMetadata(
		constants.FileKindFile,
		"application/pdf",
		"empty.pdf",
		0,
		nil,
		entities.RawMetadata{},
	)
	if meta.WidthBytes != 0 || meta.HeightBytes != 0 {
		t.Fatalf("expected zero width/height defaults, got %d/%d", meta.WidthBytes, meta.HeightBytes)
	}
	if meta.HasPassword {
		t.Fatalf("expected has_password default false")
	}
	if meta.PageCount != 0 {
		t.Fatalf("expected page_count default zero, got %d", meta.PageCount)
	}
}

func TestResolveUploadProvider_fallbackLocalWhenKindUnknown(t *testing.T) {
	provider := pkgmedia.ResolveUploadProvider(constants.FileKindFile, false)
	if provider != constants.FileProviderLocal {
		t.Fatalf("expected local fallback when kind is unknown, got %q", provider)
	}
}
