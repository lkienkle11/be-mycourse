package middleware

// HTTP headers for request correlation (structured logging, tracing).
const (
	// HeaderRequestID is the inbound correlation id (client-provided or generated).
	HeaderRequestID = "X-Request-ID"
	// HeaderCSRFToken is the header that must mirror CookieCSRFToken on unsafe methods.
	HeaderCSRFToken = "X-CSRF-Token"
)

// GinContextKeyRequestID is the Gin context key (string) for request_id.
const GinContextKeyRequestID = "request_id"

// CookieCSRFToken is the cookie used by double-submit CSRF protection.
const CookieCSRFToken = "csrf_token"
