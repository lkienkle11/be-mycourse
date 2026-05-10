package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// RBAC validation and configuration sentinels (services/rbac).
var (
	ErrRBACDatabaseNotConfigured         = stderrors.New(constants.MsgRBACDatabaseNotConfigured)
	ErrRBACInvalidUserID                 = stderrors.New(constants.MsgRBACInvalidUserID)
	ErrRBACPermissionIDRequired          = stderrors.New(constants.MsgRBACPermissionIDRequired)
	ErrRBACUserAndPermissionNameRequired = stderrors.New(constants.MsgRBACUserAndPermissionNameRequired)
	ErrRBACRoleNameRequired              = stderrors.New(constants.MsgRBACRoleNameRequired)
	ErrRBACUnknownPermissionID           = stderrors.New(constants.MsgRBACUnknownPermissionID)
	ErrRBACPermissionIDTooLong           = stderrors.New(constants.MsgRBACPermissionIDTooLong)
	ErrRBACPermissionNameRequired        = stderrors.New(constants.MsgRBACPermissionNameRequired)
	ErrRBACPermissionNameTooLong         = stderrors.New(constants.MsgRBACPermissionNameTooLong)
)
