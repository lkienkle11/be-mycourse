// Package httperr provides Gin middleware similar in intent to Spring @ControllerAdvice:
// register Middleware and Recovery on the engine, use httperr.Abort(c, err) in handlers
// (or c.Error(err); c.Abort()) so JSON responses are produced after the handler returns.
package httperr

import (
	"errors"
	"fmt"
	"net/http"

	"mycourse-io-be/pkg/errcode"
)

// HTTPError is an application error with HTTP status and a numeric application error_code.
type HTTPError struct {
	Status  int    // HTTP status for the response line
	AppCode int    // business error_code in JSON body
	Code    string // optional machine key, e.g. "bad_request" (JSON field error_key)
	Message string
	Err     error
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return http.StatusText(e.Status)
}

func (e *HTTPError) Unwrap() error { return e.Err }

// New builds an HTTPError. If message is empty, DefaultMessage(appCode) is used.
func New(status, appCode int, errorKey, message string, cause error) *HTTPError {
	if message == "" {
		message = errcode.DefaultMessage(appCode)
	}
	return &HTTPError{Status: status, AppCode: appCode, Code: errorKey, Message: message, Err: cause}
}

func BadRequest(message string) *HTTPError {
	return New(http.StatusBadRequest, errcode.BadRequest, "bad_request", message, nil)
}

func Unauthorized(message string) *HTTPError {
	return New(http.StatusUnauthorized, errcode.Unauthorized, "unauthorized", message, nil)
}

func Forbidden(message string) *HTTPError {
	return New(http.StatusForbidden, errcode.Forbidden, "forbidden", message, nil)
}

func NotFound(message string) *HTTPError {
	return New(http.StatusNotFound, errcode.NotFound, "not_found", message, nil)
}

func Conflict(message string) *HTTPError {
	return New(http.StatusConflict, errcode.Conflict, "conflict", message, nil)
}

func TooManyRequests(message string) *HTTPError {
	return New(http.StatusTooManyRequests, errcode.TooManyRequests, "too_many_requests", message, nil)
}

func Internal(message string, cause error) *HTTPError {
	return New(http.StatusInternalServerError, errcode.InternalError, "internal_error", message, cause)
}

// AsHTTPError returns (*HTTPError, true) if err unwraps to HTTPError.
func AsHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}

// Errorf wraps fmt.Errorf as internal server error (avoid for client-facing messages).
func Errorf(format string, args ...interface{}) *HTTPError {
	return New(http.StatusInternalServerError, errcode.InternalError, "internal_error", errcode.DefaultMessage(errcode.InternalError), fmt.Errorf(format, args...))
}
