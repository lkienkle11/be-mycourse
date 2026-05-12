package errors

import (
	stderrors "errors"

	"mycourse-io-be/internal/shared/constants"
)

// Local signed URL / object-key token decode errors (pkg/media/local_url_codec.go).
var (
	ErrLocalMediaTokenInvalidFormat    = stderrors.New(constants.MsgLocalMediaTokenInvalidFormat)
	ErrLocalMediaTokenInvalidPayload   = stderrors.New(constants.MsgLocalMediaTokenInvalidPayload)
	ErrLocalMediaTokenInvalidSignature = stderrors.New(constants.MsgLocalMediaTokenInvalidSignature)
	ErrLocalMediaTokenInvalid          = stderrors.New(constants.MsgLocalMediaTokenInvalid)
)
