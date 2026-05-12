package delivery_test

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	mediadelivery "mycourse-io-be/internal/media/delivery"
	mediadomain "mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
	"mycourse-io-be/internal/shared/constants"
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

	req, err := mediadelivery.BindCreateFileMultipart(c)
	if err != nil {
		t.Fatalf("bind should accept metadata payload for backward-compat: %v", err)
	}
	// In the DDD model, Kind is server-owned — CreateFileInput has no Kind field.
	// ObjectKey is preserved from the form value.
	if req.ObjectKey != "abc" {
		t.Fatalf("expected object_key=abc, got %q", req.ObjectKey)
	}
}

func TestBuildTypedMetadata_usesDefaultValuesWhenUnavailable(t *testing.T) {
	meta := mediainfra.BuildTypedMetadata(
		constants.FileKindFile,
		"application/pdf",
		"empty.pdf",
		0,
		nil,
		mediadomain.RawMetadata{},
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
	provider := mediainfra.ResolveUploadProvider(constants.FileKindFile, false)
	if provider != constants.FileProviderLocal {
		t.Fatalf("expected local fallback when kind is unknown, got %q", provider)
	}
}
