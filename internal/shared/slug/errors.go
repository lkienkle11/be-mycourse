package slug

import "errors"

var (
	// ErrInvalidFormat is returned when a slug does not match the shared slug
	// format or exceeds the caller's maximum length.
	ErrInvalidFormat = errors.New("slug has an invalid format")
	// ErrEmpty is returned when a slug is explicitly supplied but empty or
	// whitespace-only where an empty value is not meaningful (update paths).
	ErrEmpty = errors.New("slug must not be empty")
	// ErrReserved is returned when a manually supplied slug is a reserved word.
	ErrReserved = errors.New("slug is reserved")
)

// ConflictError is returned by ResolveCreate when a manually supplied slug is
// already taken. RecommendedSlug was free at the time it was generated; every
// write path still re-checks at write time and never trusts it as permanent.
// Features expose it under their own domain name (a type alias) so handlers
// can map it with errors.As to the shared 409 response.
type ConflictError struct {
	RecommendedSlug string
}

func (e *ConflictError) Error() string { return "slug already exists" }
