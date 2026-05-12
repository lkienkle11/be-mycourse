package domain

// Bunny Stream webhook HTTP header names and signature constants.
const (
	BunnyWebhookSignatureVersionHeader   = "X-BunnyStream-Signature-Version"
	BunnyWebhookSignatureAlgorithmHeader = "X-BunnyStream-Signature-Algorithm"
	BunnyWebhookSignatureHeader          = "X-BunnyStream-Signature"
	BunnyWebhookSignatureVersionV1       = "v1"
	BunnyWebhookSignatureAlgorithmHMAC   = "hmac-sha256"
)
