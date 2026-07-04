package errors

import (
	"errors"

	"mycourse-io-be/internal/shared/constants"
)

// Media bounded-context sentinels.
var (
	ErrMediaOptimisticLock            = errors.New(constants.MsgMediaOptimisticLockConflict)
	ErrMediaReuseMismatch             = errors.New(constants.MsgMediaReuseMismatch)
	ErrExecutableUploadRejected       = errors.New(constants.MsgExecutableUploadRejected)
	ErrImageEncodeBusy                = errors.New(constants.MsgImageEncodeBusy)
	ErrDependencyNotConfigured        = errors.New(constants.MsgMediaDependencyNotConfigured)
	ErrMediaVideoGUIDRequired         = errors.New(constants.MsgMediaVideoGUIDRequired)
	ErrMediaObjectKeyRequired         = errors.New(constants.MsgMediaObjectKeyRequired)
	ErrMediaFileNotFoundForObjectKey  = errors.New(constants.MsgMediaFileNotFoundForObjectKey)
	ErrMediaAccessDenied              = errors.New(constants.MsgMediaAccessDenied)
	ErrBunnyStreamResponseMissingGUID = errors.New(constants.MsgBunnyStreamResponseMissingVideoGUID)
)

// RBAC bounded-context sentinels.
var (
	ErrRBACDatabaseNotConfigured         = errors.New(constants.MsgRBACDatabaseNotConfigured)
	ErrRBACInvalidUserID                 = errors.New(constants.MsgRBACInvalidUserID)
	ErrRBACPermissionIDRequired          = errors.New(constants.MsgRBACPermissionIDRequired)
	ErrRBACUserAndPermissionNameRequired = errors.New(constants.MsgRBACUserAndPermissionNameRequired)
	ErrRBACRoleNameRequired              = errors.New(constants.MsgRBACRoleNameRequired)
	ErrRBACUnknownPermissionID           = errors.New(constants.MsgRBACUnknownPermissionID)
	ErrRBACPermissionIDTooLong           = errors.New(constants.MsgRBACPermissionIDTooLong)
	ErrRBACPermissionNameRequired        = errors.New(constants.MsgRBACPermissionNameRequired)
	ErrRBACPermissionNameTooLong         = errors.New(constants.MsgRBACPermissionNameTooLong)
)
