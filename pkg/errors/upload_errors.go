package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// ErrFileExceedsMaxUploadSize is returned when a single uploaded file exceeds constants.MaxMediaUploadFileBytes
// (declared size, buffered read cap, or stream longer than the limit).
// Same text as pkg/errcode default for FileTooLarge — constants.MsgFileTooLargeUpload (do not duplicate).
var ErrFileExceedsMaxUploadSize = stderrors.New(constants.MsgFileTooLargeUpload)
