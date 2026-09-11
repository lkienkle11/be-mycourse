package requestprincipal

import "context"

type Type string

const TypeUser Type = "USER"

type Principal struct {
	Type              Type
	ID                string
	GlobalPermissions map[string]struct{}
}

type contextKey struct{}

func WithContext(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func FromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}
