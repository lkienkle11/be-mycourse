package slug

import "strings"

// ParseManual normalizes an optional manual slug supplied on create. An empty
// or whitespace-only value means "no manual slug" and returns ("", nil). Any
// other value must satisfy ValidateManualFormat or ErrInvalidFormat is
// returned. Callers map the sentinel to their own validation error.
func ParseManual(raw string, maxLen int) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	trimmed, ok := ValidateManualFormat(raw, maxLen)
	if !ok {
		return "", ErrInvalidFormat
	}
	return trimmed, nil
}

// ParseUpdateInput normalizes a PATCH slug field given as the
// validate.Optional[string] pair (set, value) - taken as plain values so this
// package stays free of the validate/utils import chain that internal/shared/response
// depends on. set=false (omitted) returns (nil, nil) meaning "leave unchanged".
// An explicit null, empty or whitespace-only value returns ErrEmpty; a
// malformed value returns ErrInvalidFormat.
func ParseUpdateInput(set bool, value string, maxLen int) (*string, error) {
	if !set {
		return nil, nil
	}
	if strings.TrimSpace(value) == "" {
		return nil, ErrEmpty
	}
	trimmed, ok := ValidateManualFormat(value, maxLen)
	if !ok {
		return nil, ErrInvalidFormat
	}
	return &trimmed, nil
}
