package media

import (
	"errors"

	"mycourse-io-be/constants"
)

// ErrFileExceedsMaxUploadSize is returned when a single uploaded file exceeds constants.MaxMediaUploadFileBytes
// (declared size, buffered read cap, or stream longer than the limit).
// Same text as pkg/errcode default for FileTooLarge — constants.MsgFileTooLargeUpload (do not duplicate).
var ErrFileExceedsMaxUploadSize = errors.New(constants.MsgFileTooLargeUpload)
