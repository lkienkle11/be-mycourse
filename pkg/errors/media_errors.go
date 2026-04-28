package errors

import (
	"errors"

	"mycourse-io-be/constants"
)

var (
	ErrMediaOptimisticLock = errors.New(constants.MsgMediaOptimisticLockConflict)
	ErrMediaReuseMismatch  = errors.New(constants.MsgMediaReuseMismatch)
)
