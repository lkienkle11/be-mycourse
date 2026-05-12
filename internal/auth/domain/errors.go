package domain

import (
	stderrors "errors"
	"fmt"
)

// Auth domain sentinel errors — compare with errors.Is.
var (
	ErrEmailAlreadyExists          = stderrors.New("email already registered")
	ErrInvalidCredentials          = stderrors.New("invalid email or password")
	ErrWeakPassword                = stderrors.New("password does not meet requirements")
	ErrEmailNotConfirmed           = stderrors.New("email not confirmed")
	ErrUserDisabled                = stderrors.New("user account is disabled")
	ErrInvalidConfirmToken         = stderrors.New("invalid or expired confirmation token")
	ErrUserNotFound                = stderrors.New("user not found")
	ErrInvalidSession              = stderrors.New("invalid session")
	ErrRefreshTokenExpired         = stderrors.New("refresh token expired")
	ErrRegistrationAbandoned       = stderrors.New("registration was abandoned because confirmation email limits were exceeded")
	ErrConfirmationEmailSendFailed = stderrors.New("confirmation email could not be sent; please try again later")
)

// RegistrationEmailRateLimitedError is returned when the Redis sliding window for
// registration confirmation emails is exceeded. Handlers map to HTTP 429 + errcode 4010.
type RegistrationEmailRateLimitedError struct {
	RetryAfterSeconds int64
}

func (e *RegistrationEmailRateLimitedError) Error() string {
	return fmt.Sprintf("too many confirmation emails were sent recently; please try again later (retry_after_seconds=%d)", e.RetryAfterSeconds)
}
