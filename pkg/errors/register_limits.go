package errors

import (
	"fmt"

	"mycourse-io-be/constants"
)

// RegistrationEmailRateLimitedError is returned when the Redis sliding window for
// registration confirmation emails is exceeded. Handlers map it to HTTP 429 and errcode 4010.
// Typed errors for auth limits belong in pkg/errors, not pkg/entities (entities are pure data).
type RegistrationEmailRateLimitedError struct {
	RetryAfterSeconds int64
}

func (e *RegistrationEmailRateLimitedError) Error() string {
	return fmt.Sprintf("%s (retry_after_seconds=%d)", constants.MsgAuthRegistrationEmailRateLimited, e.RetryAfterSeconds)
}
