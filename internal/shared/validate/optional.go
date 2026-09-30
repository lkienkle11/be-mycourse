package validate

import (
	"bytes"
	"encoding/json"
)

// Optional distinguishes "field omitted from the request body" from "field
// explicitly present" (including an explicit JSON null) when decoding a
// PATCH-style request — something a plain pointer field cannot do:
// encoding/json leaves a *T field nil for both an omitted key and an
// explicit `null` value (verified directly against encoding/json, not
// assumed), so any endpoint needing "omitted = leave unchanged, explicit
// null = reject" semantics must use this instead of *T.
type Optional[T any] struct {
	Value T
	Set   bool
}

// UnmarshalJSON is only invoked by encoding/json (and anything built on it,
// e.g. Gin's ShouldBindJSON) when the field's key is present in the source
// object — including when its value is the literal null. An absent key never
// calls this at all, leaving Set at its zero value (false).
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		var zero T
		o.Value = zero
		return nil
	}
	return json.Unmarshal(data, &o.Value)
}
