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
	ErrUserBanned                  = stderrors.New("user account is temporarily banned")
	ErrInvalidConfirmToken         = stderrors.New("invalid or expired confirmation token")
	ErrUserNotFound                = stderrors.New("user not found")
	ErrInvalidSession              = stderrors.New("invalid session")
	ErrRefreshTokenExpired         = stderrors.New("refresh token expired")
	ErrRegistrationAbandoned       = stderrors.New("registration was abandoned because confirmation email limits were exceeded")
	ErrConfirmationEmailSendFailed = stderrors.New("confirmation email could not be sent; please try again later")

	// OAuth — Google (mapped to 4013/4014/4015 in delivery).
	ErrInvalidGoogleCode      = stderrors.New("invalid google authorization code")
	ErrGoogleEmailNotVerified = stderrors.New("google email not verified")
	ErrOAuthIdentityConflict  = stderrors.New("oauth identity conflict")

	// OAuth — X (mapped to 4016/4017/4019 in delivery). 4018 is FE-local only.
	ErrInvalidXCode         = stderrors.New("invalid x authorization code")
	ErrXEmailUnavailable    = stderrors.New("x account has no usable email")
	ErrXAccountLinkRequired = stderrors.New("x account requires linking to an existing email account")

	// OAuth — Discord (mapped to 4023/4024/4025 in delivery).
	ErrInvalidDiscordCode      = stderrors.New("invalid discord authorization code")
	ErrDiscordEmailNotVerified = stderrors.New("discord email not verified")
	ErrDiscordEmailUnavailable = stderrors.New("discord account has no usable email")
)

// RegistrationEmailRateLimitedError is returned when the Redis sliding window for
// registration confirmation emails is exceeded. Handlers map to HTTP 429 + errcode 4010.
type RegistrationEmailRateLimitedError struct {
	RetryAfterSeconds int64
}

func (e *RegistrationEmailRateLimitedError) Error() string {
	return fmt.Sprintf("too many confirmation emails were sent recently; please try again later (retry_after_seconds=%d)", e.RetryAfterSeconds)
}
