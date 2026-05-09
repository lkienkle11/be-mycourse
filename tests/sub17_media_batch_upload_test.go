package tests

import (
	"errors"
	"mime/multipart"
	"testing"

	"mycourse-io-be/constants"
	pkgerrors "mycourse-io-be/pkg/errors"
	pkgmedia "mycourse-io-be/pkg/media"
	mediaservice "mycourse-io-be/services/media"
)

func TestValidateMultipartFileHeaders_zeroParts(t *testing.T) {
	err := pkgmedia.ValidateMultipartFileHeaders(nil)
	if !errors.Is(err, pkgerrors.ErrMediaFilesRequired) {
		t.Fatalf("got %v", err)
	}
	err = pkgmedia.ValidateMultipartFileHeaders([]*multipart.FileHeader{})
	if !errors.Is(err, pkgerrors.ErrMediaFilesRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateMultipartFileHeaders_tooManyParts(t *testing.T) {
	h := make([]*multipart.FileHeader, constants.MaxMediaFilesPerRequest+1)
	for i := range h {
		h[i] = &multipart.FileHeader{Filename: "x"}
	}
	err := pkgmedia.ValidateMultipartFileHeaders(h)
	if !errors.Is(err, pkgerrors.ErrMediaTooManyFilesInRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateMultipartFileHeaders_declaredTotalExceedsTwoGiB(t *testing.T) {
	h := []*multipart.FileHeader{
		{Filename: "a", Size: constants.MaxMediaUploadFileBytes},
		{Filename: "b", Size: 1},
	}
	err := pkgmedia.ValidateMultipartFileHeaders(h)
	if !errors.Is(err, pkgerrors.ErrMediaMultipartTotalTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateMultipartFileHeaders_declaredOKAtBoundary(t *testing.T) {
	h := []*multipart.FileHeader{
		{Filename: "a", Size: constants.MaxMediaMultipartTotalBytes},
	}
	if err := pkgmedia.ValidateMultipartFileHeaders(h); err != nil {
		t.Fatal(err)
	}
}

func TestParallelUploadProbeHookAvailable(t *testing.T) {
	prev := mediaservice.MediaUploadParallelStartProbe
	defer func() { mediaservice.MediaUploadParallelStartProbe = prev }()
	mediaservice.MediaUploadParallelStartProbe = func() {}
	if constants.MaxConcurrentMediaUploadWorkers < 1 {
		t.Fatal("MaxConcurrentMediaUploadWorkers must be >= 1")
	}
}
