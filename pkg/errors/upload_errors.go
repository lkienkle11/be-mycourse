package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// ErrFileExceedsMaxUploadSize is returned when a single uploaded file exceeds constants.MaxMediaUploadFileBytes
// (declared size, buffered read cap, or stream longer than the limit).
// Same text as pkg/errcode default for FileTooLarge — constants.MsgFileTooLargeUpload (do not duplicate).
var ErrFileExceedsMaxUploadSize = stderrors.New(constants.MsgFileTooLargeUpload)

// ErrMediaMultipartTotalTooLarge is returned when combined parts exceed constants.MaxMediaMultipartTotalBytes.
var ErrMediaMultipartTotalTooLarge = stderrors.New(constants.MsgMediaMultipartTotalTooLarge)

// ErrMediaTooManyFilesInRequest is returned when more than constants.MaxMediaFilesPerRequest parts are present.
var ErrMediaTooManyFilesInRequest = stderrors.New(constants.MsgMediaTooManyFilesInRequest)

// ErrMediaFilesRequired is returned when no file parts were submitted for create/update.
var ErrMediaFilesRequired = stderrors.New(constants.MsgMediaFilesRequired)

// ErrMediaBatchDeleteTooManyIDs is returned when batch delete lists too many keys.
var ErrMediaBatchDeleteTooManyIDs = stderrors.New(constants.MsgMediaBatchDeleteTooManyIDs)

// ErrMediaDuplicateObjectKeysInBatchDelete is returned when batch delete payload lists duplicate keys.
var ErrMediaDuplicateObjectKeysInBatchDelete = stderrors.New(constants.MsgMediaDuplicateObjectKeysInBatchDelete)

// ErrBatchDeleteEmptyKeys is returned when DeleteFilesByObjectKeys receives an empty slice.
var ErrBatchDeleteEmptyKeys = stderrors.New(constants.MsgBatchDeleteEmptyObjectKeys)
