package errcode

// DefaultMessage returns the canonical description for an application error code.
// Unknown codes fall back to Unknown (9999): "Unknown Error".
func DefaultMessage(code int) string {
	if m, ok := defaultMessages[code]; ok {
		return m
	}
	return defaultMessages[Unknown]
}

var defaultMessages = map[int]string{
	Unknown:          "Unknown Error",
	InvalidJSON:      "Request body is not valid JSON",
	ValidationFailed: "Validation failed",
	ValidationField:  "Field validation failed",

	BadRequest:      "Bad request",
	Unauthorized:    "Unauthorized",
	Forbidden:       "Forbidden",
	NotFound:        "Resource not found",
	Conflict:        "Conflict",
	TooManyRequests: "Too many requests",

	InternalError: "Internal server error",
	Panic:         "Internal server error",
}
