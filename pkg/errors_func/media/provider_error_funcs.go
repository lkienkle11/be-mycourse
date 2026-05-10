// Package errfuncmedia holds error helper functions for B2/Bunny provider paths (Rule 19).
package errfuncmedia

import (
	stderrors "errors"
	"net/http"

	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
)

// HTTPStatusForProviderCode maps provider errcodes to HTTP status.
func HTTPStatusForProviderCode(code int) int {
	switch code {
	case errcode.BunnyCreateFailed, errcode.BunnyUploadFailed, errcode.BunnyInvalidResponse:
		return http.StatusBadGateway
	case errcode.BunnyVideoNotFound:
		return http.StatusNotFound
	case errcode.BunnyGetVideoFailed:
		return http.StatusBadGateway
	case errcode.ImageEncodeBusy:
		return http.StatusServiceUnavailable // 503
	default:
		return http.StatusInternalServerError
	}
}

// AsProviderError unwraps *pkgerrors.ProviderError.
func AsProviderError(err error) (*pkgerrors.ProviderError, bool) {
	var pe *pkgerrors.ProviderError
	if stderrors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}
