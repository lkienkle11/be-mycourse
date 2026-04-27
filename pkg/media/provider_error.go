package media

import (
	"errors"
	"net/http"

	"mycourse-io-be/pkg/errcode"
)

// ProviderError carries an application errcode for B2/Bunny client failures.
type ProviderError struct {
	Code int
	Msg  string
	Err  error
}

func (e *ProviderError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return errcode.DefaultMessage(e.Code)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// HTTPStatusForProviderCode maps provider errcodes to HTTP status.
func HTTPStatusForProviderCode(code int) int {
	switch code {
	case errcode.BunnyCreateFailed, errcode.BunnyUploadFailed, errcode.BunnyInvalidResponse:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// AsProviderError unwraps *ProviderError.
func AsProviderError(err error) (*ProviderError, bool) {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}
