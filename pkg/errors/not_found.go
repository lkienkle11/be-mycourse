package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// ErrNotFound means the requested persisted row or aggregate does not exist.
// Handlers in api/ should compare with this sentinel instead of importing gorm.io/gorm.
var ErrNotFound = stderrors.New(constants.MsgNotFound)
