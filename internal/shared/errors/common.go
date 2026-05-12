package errors

import (
	stderrors "errors"

	"mycourse-io-be/internal/shared/constants"
)

// ErrNilDatabase is returned when a DB handle is required but nil was passed.
var ErrNilDatabase = stderrors.New(constants.MsgNilDatabase)
