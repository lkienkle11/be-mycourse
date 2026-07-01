package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/setting"
)

func TestR2ObjectContentType_prefersMetadataMIME(t *testing.T) {
	meta := domain.RawMetadata{domain.MediaMetaKeyMimeType: "image/webp"}
	if got := r2ObjectContentType(meta); got != "image/webp" {
		t.Fatalf("content type = %q, want image/webp", got)
	}
}

func TestR2ObjectContentType_rejectsBlockedMetadataMIME(t *testing.T) {
	meta := domain.RawMetadata{domain.MediaMetaKeyMimeType: "text/html"}
	if got := r2ObjectContentType(meta); got != "application/octet-stream" {
		t.Fatalf("content type = %q, want application/octet-stream", got)
	}
}

func TestR2ObjectContentType_defaultsToOctetStream(t *testing.T) {
	if got := r2ObjectContentType(nil); got != "application/octet-stream" {
		t.Fatalf("content type = %q, want application/octet-stream", got)
	}
}

type recordingR2Putter struct {
	lastInput *s3.PutObjectInput
}

func (r *recordingR2Putter) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	r.lastInput = input
	return &s3.PutObjectOutput{}, nil
}

func TestUploadR2WithPutter_setsPutObjectContentType(t *testing.T) {
	oldSetting := setting.MediaSetting
	t.Cleanup(func() { setting.MediaSetting = oldSetting })

	setting.MediaSetting = &setting.Media{R2: setting.MediaR2Storage{Bucket: "test-bucket"}}
	putter := &recordingR2Putter{}
	clients := &CloudClients{R2BucketName: "test-bucket"}
	meta := domain.RawMetadata{domain.MediaMetaKeyMimeType: "image/webp"}

	_, err := uploadR2WithPutter(
		clients,
		putter,
		context.Background(),
		"12345678-photo.webp",
		"photo.webp",
		strings.NewReader("payload"),
		meta,
	)
	if err != nil {
		t.Fatalf("uploadR2WithPutter returned error: %v", err)
	}
	if putter.lastInput == nil {
		t.Fatal("expected PutObject to be called")
	}
	if got := awsStringValue(putter.lastInput.ContentType); got != "image/webp" {
		t.Fatalf("PutObject ContentType = %q, want image/webp", got)
	}
}

func awsStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
