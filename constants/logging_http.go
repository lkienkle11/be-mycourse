package constants

// HTTP headers for request correlation (structured logging, tracing).
const (
	// HeaderRequestID is the inbound correlation id (client-provided or generated).
	HeaderRequestID = "X-Request-ID"
)

// Gin context key (string) for request_id — same value as context key semantics in pkg/logger.
const GinContextKeyRequestID = "request_id"
