// Package errcode defines numeric application error_code values and their default messages
// (see messages.go). HTTP status remains on the response line; error_code is in the JSON body.
// Shared literals used both here (default JSON message) and in errors.New sentinels must be defined
// only in constants/error_msg.go and referenced from messages.go (e.g. FileTooLarge + MsgFileTooLargeUpload).
package errcode

// Application error codes (numeric), in addition to HTTP status on the response line.
// Default / fallback: Unknown (9999) → DefaultMessage returns "Unknown Error".
// Use Success (0) for all successful responses.

const (
	Success = 0 // operation completed successfully
	Unknown = 9999

	// Transport / parsing (1xxx)
	InvalidJSON = 1001

	// Validation (2xxx)
	ValidationFailed = 2001
	ValidationField  = 2002 // used per-field in details when applicable
	FileTooLarge     = 2003 // single-file upload exceeds cap; default message = constants.MsgFileTooLargeUpload (see messages.go)

	// Client / HTTP-shaped (3xxx) — align loosely with HTTP family
	BadRequest      = 3001
	Unauthorized    = 3002
	Forbidden       = 3003
	NotFound        = 3004
	Conflict        = 3005
	TooManyRequests = 3006

	// Auth (4xxx)
	EmailAlreadyExists  = 4001
	InvalidCredentials  = 4002
	WeakPassword        = 4003
	EmailNotConfirmed   = 4004
	UserDisabled        = 4005
	InvalidConfirmToken = 4006
	InvalidSession      = 4007
	RefreshTokenExpired = 4008

	// Server (9xxx)
	InternalError = 9001
	Panic         = 9998

	// Media upstream (90xx)
	B2BucketNotConfigured    = 9010
	BunnyStreamNotConfigured = 9011
	BunnyCreateFailed        = 9012
	BunnyUploadFailed        = 9013
	BunnyInvalidResponse     = 9014
	BunnyVideoNotFound       = 9015
	BunnyGetVideoFailed      = 9016
)
