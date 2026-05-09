package media

import (
	"mime/multipart"

	"mycourse-io-be/constants"
	pkgerrors "mycourse-io-be/pkg/errors"
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
		return pkgerrors.ErrMediaFilesRequired
	}
	if len(headers) > constants.MaxMediaFilesPerRequest {
		return pkgerrors.ErrMediaTooManyFilesInRequest
	}
	var sum int64
	allKnown := true
	for _, h := range headers {
		if h.Size < 0 {
			allKnown = false
			break
		}
		if h.Size > constants.MaxMediaUploadFileBytes {
			return pkgerrors.ErrFileExceedsMaxUploadSize
		}
		sum += h.Size
	}
	if allKnown && sum > constants.MaxMediaMultipartTotalBytes {
		return pkgerrors.ErrMediaMultipartTotalTooLarge
	}
	return nil
}
