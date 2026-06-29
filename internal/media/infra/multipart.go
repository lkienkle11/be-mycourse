package infra

import (
	"io"
	"mime/multipart"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
)

// CollectMultipartFileHeaders returns headers for field "files", or legacy "file", after multipart parse.
func CollectMultipartFileHeaders(form *multipart.Form) []*multipart.FileHeader {
	if form == nil {
		return nil
	}
	f := form.File["files"]
	if len(f) > 0 {
		return f
	}
	return form.File["file"]
}

// ValidateMultipartFileHeaders enforces count and declared-size caps before streaming bodies.
func ValidateMultipartFileHeaders(headers []*multipart.FileHeader) error {
	if len(headers) == 0 {
		return apperrors.ErrMediaFilesRequired
	}
	if len(headers) > constants.MaxMediaFilesPerRequest {
		return apperrors.ErrMediaTooManyFilesInRequest
	}
	var sum int64
	allKnown := true
	for _, h := range headers {
		if h.Size < 0 {
			allKnown = false
			break
		}
		if h.Size > constants.MaxMediaUploadFileBytes {
			return apperrors.ErrFileExceedsMaxUploadSize
		}
		sum += h.Size
	}
	if allKnown && sum > constants.MaxMediaMultipartTotalBytes {
		return apperrors.ErrMediaMultipartTotalTooLarge
	}
	return nil
}

// CloseOpenedUploadParts closes all opened streams (best-effort).
func CloseOpenedUploadParts(parts []domain.OpenedUploadPart) {
	for _, p := range parts {
		if p.File != nil {
			_ = p.File.Close()
		}
	}
}

// OpenUploadParts opens each header; on error closes already-opened streams.
func OpenUploadParts(headers []*multipart.FileHeader) ([]domain.OpenedUploadPart, error) {
	parts := make([]domain.OpenedUploadPart, 0, len(headers))
	for _, h := range headers {
		f, err := h.Open()
		if err != nil {
			CloseOpenedUploadParts(parts)
			return nil, err
		}
		parts = append(parts, domain.OpenedUploadPart{File: f, Header: h})
	}
	return parts, nil
}

// DrainDiscard closes r after discarding any unread bytes (multipart cleanup helper).
func DrainDiscard(r io.Closer) {
	if r == nil {
		return
	}
	if rc, ok := r.(io.ReadCloser); ok {
		_, _ = io.Copy(io.Discard, rc)
	}
	_ = r.Close()
}
