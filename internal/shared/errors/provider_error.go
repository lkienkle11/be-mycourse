package errors

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
	return DefaultMessage(e.Code)
}

func (e *ProviderError) Unwrap() error { return e.Err }
