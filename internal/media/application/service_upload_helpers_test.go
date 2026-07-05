package application

import (
	"testing"

	mediainfra "mycourse-io-be/internal/media/infra"
	"mycourse-io-be/internal/shared/constants"
)

func TestPrepareNormalizedUploadPart_routesVideoFromDetectedMIME(t *testing.T) {
	gw := mediainfra.NewStorageGateway()
	payload := []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	got, err := prepareNormalizedUploadPart(gw, "upload.bin", "application/octet-stream", payload, "", "")
	if err != nil {
		t.Fatalf("prepareNormalizedUploadPart returned error: %v", err)
	}
	if got.kind != constants.FileKindVideo {
		t.Fatalf("kind = %q, want %q", got.kind, constants.FileKindVideo)
	}
}

func TestPrepareNormalizedUploadPart_encodesImageFromDetectedMIME(t *testing.T) {
	gw := mediainfra.NewStorageGateway()
	payload := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0, 0, 0, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0,
		0x90, 0x77, 0x53, 0xde,
		0, 0, 0, 0x0c, 0x49, 0x44, 0x41, 0x54,
		8, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f, 0, 5, 0xfe, 0x2, 0xfe, 0xdc, 0xcc,
		0x59, 0xe7, 0, 0, 0, 0, 0, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	got, err := prepareNormalizedUploadPart(gw, "photo.bin", "application/octet-stream", payload, "", "")
	if err != nil {
		t.Fatalf("prepareNormalizedUploadPart returned error: %v", err)
	}
	if got.mime != "image/webp" {
		t.Fatalf("mime = %q, want image/webp after encode", got.mime)
	}
}
