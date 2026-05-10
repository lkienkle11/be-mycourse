package constants

// Headers returned on HTTP 429 when registration confirmation email hits the Redis window.
const (
	HeaderRegisterRetryAfter         = "Retry-After"
	HeaderRegisterRetryAfterExtended = "X-Mycourse-Register-Retry-After"
)
