package domain

// ProviderError carries an application errcode for cloud client failures.
type ProviderError struct {
	Code int
	Msg  string
	Err  error
}

func (e *ProviderError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "provider error"
}

func (e *ProviderError) Unwrap() error { return e.Err }
