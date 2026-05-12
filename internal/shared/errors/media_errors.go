package errors

import (
	"errors"

	"mycourse-io-be/internal/shared/constants"
)

var (
	ErrMediaOptimisticLock            = errors.New(constants.MsgMediaOptimisticLockConflict)
	ErrMediaReuseMismatch             = errors.New(constants.MsgMediaReuseMismatch)
	ErrExecutableUploadRejected       = errors.New(constants.MsgExecutableUploadRejected)
	ErrImageEncodeBusy                = errors.New(constants.MsgImageEncodeBusy)
	ErrDependencyNotConfigured        = errors.New(constants.MsgMediaDependencyNotConfigured)
	ErrMediaVideoGUIDRequired         = errors.New(constants.MsgMediaVideoGUIDRequired)
	ErrMediaObjectKeyRequired         = errors.New(constants.MsgMediaObjectKeyRequired)
	ErrMediaFileNotFoundForObjectKey  = errors.New(constants.MsgMediaFileNotFoundForObjectKey)
	ErrBunnyStreamResponseMissingGUID = errors.New(constants.MsgBunnyStreamResponseMissingVideoGUID)
)
