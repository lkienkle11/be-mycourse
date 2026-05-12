package middleware

// HTTP headers for request correlation (structured logging, tracing).
const (
	// HeaderRequestID is the inbound correlation id (client-provided or generated).
	HeaderRequestID = "X-Request-ID"
)

// GinContextKeyRequestID is the Gin context key (string) for request_id.
const GinContextKeyRequestID = "request_id"
