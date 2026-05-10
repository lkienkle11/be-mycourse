package errors

import (
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
