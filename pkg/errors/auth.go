package errors

import (
	stderrors "errors"

	"mycourse-io-be/constants"
)

// Auth register/login/confirm/refresh session sentinels (returned by services/auth; compare with errors.Is against these vars).
var (
	ErrEmailAlreadyExists          = stderrors.New(constants.MsgAuthEmailAlreadyExists)
	ErrInvalidCredentials          = stderrors.New(constants.MsgAuthInvalidCredentials)
	ErrWeakPassword                = stderrors.New(constants.MsgAuthWeakPassword)
	ErrEmailNotConfirmed           = stderrors.New(constants.MsgAuthEmailNotConfirmed)
	ErrUserDisabled                = stderrors.New(constants.MsgAuthUserDisabled)
	ErrInvalidConfirmToken         = stderrors.New(constants.MsgAuthInvalidConfirmToken)
	ErrUserNotFound                = stderrors.New(constants.MsgAuthUserNotFound)
	ErrInvalidSession              = stderrors.New(constants.MsgAuthInvalidSession)
	ErrRefreshTokenExpired         = stderrors.New(constants.MsgAuthRefreshTokenExpired)
	ErrRegistrationAbandoned       = stderrors.New(constants.MsgAuthRegistrationAbandoned)
	ErrConfirmationEmailSendFailed = stderrors.New(constants.MsgAuthConfirmationEmailSendFailed)
)
