package domain

import (
	"context"

	"mycourse-io-be/internal/shared/requestprincipal"
)

// WithPrincipal attaches a trusted, server-created principal to a request context.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return requestprincipal.WithContext(ctx, principal)
}

// PrincipalFromContext returns the trusted principal attached by authentication middleware.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	return requestprincipal.FromContext(ctx)
}
