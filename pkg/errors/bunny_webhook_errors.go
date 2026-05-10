package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// Bunny webhook parse/validate sentinels (handlers map to HTTP + errcode).
var (
	ErrBunnyWebhookJSONInvalid    = stderrors.New(constants.MsgBunnyWebhookJSONInvalid)
	ErrBunnyWebhookPayloadInvalid = stderrors.New(constants.MsgBunnyWebhookPayloadInvalid)
)
