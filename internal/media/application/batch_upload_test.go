package application_test

import (
	"errors"
	"mime/multipart"
	"testing"

	mediaapp "mycourse-io-be/internal/media/application"
	mediainfra "mycourse-io-be/internal/media/infra"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/constants"
)

func TestValidateMultipartFileHeaders_zeroParts(t *testing.T) {
	err := mediainfra.ValidateMultipartFileHeaders(nil)
	if !errors.Is(err, apperrors.ErrMediaFilesRequired) {
		t.Fatalf("got %v", err)
	}
	err = mediainfra.ValidateMultipartFileHeaders([]*multipart.FileHeader{})
	if !errors.Is(err, apperrors.ErrMediaFilesRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateMultipartFileHeaders_tooManyParts(t *testing.T) {
	h := make([]*multipart.FileHeader, constants.MaxMediaFilesPerRequest+1)
	for i := range h {
		h[i] = &multipart.FileHeader{Filename: "x"}
	}
	err := mediainfra.ValidateMultipartFileHeaders(h)
	if !errors.Is(err, apperrors.ErrMediaTooManyFilesInRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateMultipartFileHeaders_declaredTotalExceedsTwoGiB(t *testing.T) {
	h := []*multipart.FileHeader{
		{Filename: "a", Size: constants.MaxMediaUploadFileBytes},
		{Filename: "b", Size: 1},
	}
	err := mediainfra.ValidateMultipartFileHeaders(h)
	if !errors.Is(err, apperrors.ErrMediaMultipartTotalTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateMultipartFileHeaders_declaredOKAtBoundary(t *testing.T) {
	h := []*multipart.FileHeader{
		{Filename: "a", Size: constants.MaxMediaMultipartTotalBytes},
	}
	if err := mediainfra.ValidateMultipartFileHeaders(h); err != nil {
		t.Fatal(err)
	}
}

func TestParallelUploadProbeHookAvailable(t *testing.T) {
	prev := mediaapp.MediaUploadParallelStartProbe
	defer func() { mediaapp.MediaUploadParallelStartProbe = prev }()
	mediaapp.MediaUploadParallelStartProbe = func() {}
	if constants.MaxConcurrentMediaUploadWorkers < 1 {
		t.Fatal("MaxConcurrentMediaUploadWorkers must be >= 1")
	}
}
