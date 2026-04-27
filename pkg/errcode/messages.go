// Package errcode — numeric API codes and default JSON "message" strings per code.
//
// Rule: when a default message must match a sentinel or shared literal (e.g. upload size),
// the string lives once in constants/error_msg.go and this file references it — never duplicate the literal.
package errcode

import "mycourse-io-be/constants"

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
	FileTooLarge:     constants.MsgFileTooLargeUpload,

	BadRequest:      "Bad request",
	Unauthorized:    "Unauthorized",
	Forbidden:       "Forbidden",
	NotFound:        "Resource not found",
	Conflict:        "Conflict",
	TooManyRequests: "Too many requests",

	EmailAlreadyExists:  "Email address is already registered",
	InvalidCredentials:  "Invalid email or password",
	WeakPassword:        "Password must be at least 8 characters and contain uppercase, lowercase, and special characters",
	EmailNotConfirmed:   "Please confirm your email address before logging in",
	UserDisabled:        "Your account has been disabled",
	InvalidConfirmToken: "Invalid or expired confirmation token",
	InvalidSession:      "Invalid or missing session",
	RefreshTokenExpired: "Session has expired, please log in again",

	InternalError: "Internal server error",
	Panic:         "Internal server error",

	B2BucketNotConfigured:    constants.MsgMediaB2BucketNotConfigured,
	BunnyStreamNotConfigured: constants.MsgMediaBunnyStreamNotConfigured,
	BunnyCreateFailed:        constants.MsgMediaBunnyCreateFailed,
	BunnyUploadFailed:        constants.MsgMediaBunnyUploadFailed,
	BunnyInvalidResponse:     constants.MsgMediaBunnyInvalidResponse,
}
