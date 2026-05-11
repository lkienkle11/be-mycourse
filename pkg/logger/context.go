package logger

import (
	"context"

	"go.uber.org/zap"
)

// requestIDContextKey is a private sentinel type for context.Value (staticcheck SA1029 — avoid string keys).
type requestIDContextKey struct{}

var requestIDKey = requestIDContextKey{}

// WithRequestID returns a child context carrying request_id for FromContext.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext returns the correlation id if present.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// FromContext returns zap.L() enriched with request_id when the context carries one.
// Use in services and handlers after HTTP middleware has attached the id.
func FromContext(ctx context.Context) *zap.Logger {
	base := zap.L()
	if ctx == nil {
		return base
	}
	if id := RequestIDFromContext(ctx); id != "" {
		return base.With(zap.String("request_id", id))
	}
	return base
}
