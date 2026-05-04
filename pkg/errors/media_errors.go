package errors

import (
	"errors"

	"mycourse-io-be/constants"
)

var (
	ErrMediaOptimisticLock      = errors.New(constants.MsgMediaOptimisticLockConflict)
	ErrMediaReuseMismatch       = errors.New(constants.MsgMediaReuseMismatch)
	ErrExecutableUploadRejected = errors.New(constants.MsgExecutableUploadRejected)
	ErrImageEncodeBusy          = errors.New(constants.MsgImageEncodeBusy)
	ErrDependencyNotConfigured  = errors.New(constants.MsgMediaDependencyNotConfigured)
)
