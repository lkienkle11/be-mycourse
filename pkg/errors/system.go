package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// System privileged-user / system_app_config sentinel errors (used by services.SystemLogin, etc.).
var (
	ErrSystemAppConfigMissing = stderrors.New(constants.MsgSystemAppConfigMissing)
	ErrSystemSecretsNotReady  = stderrors.New(constants.MsgSystemSecretsNotReady)
	ErrSystemLoginFailed      = stderrors.New(constants.MsgSystemLoginFailed)
)
