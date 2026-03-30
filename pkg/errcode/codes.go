// Package errcode defines numeric application error_code values and their default messages
// (see messages.go). HTTP status remains on the response line; error_code is in the JSON body.
package errcode

// Application error codes (numeric), in addition to HTTP status on the response line.
// Default / fallback: Unknown (9999) → DefaultMessage returns "Unknown Error".

const (
	Unknown = 9999

	// Transport / parsing (1xxx)
	InvalidJSON = 1001

	// Validation (2xxx)
	ValidationFailed = 2001
	ValidationField  = 2002 // used per-field in details when applicable

	// Client / HTTP-shaped (3xxx) — align loosely with HTTP family
	BadRequest      = 3001
	Unauthorized    = 3002
	Forbidden       = 3003
	NotFound        = 3004
	Conflict        = 3005
	TooManyRequests = 3006

	// Server (9xxx)
	InternalError = 9001
	Panic         = 9998
)
