package constants

// Bunny Stream literals only (Global Constants Placement). Status enum + StatusString() live in bunny_video_status.go.

// FinishedWebhookBunnyStatus is the Bunny webhook numeric status meaning encoding finished.
const FinishedWebhookBunnyStatus = int(BunnyFinished)

// SignBunnyIFrameRegex matches signed iframe query noise for cleanup/normalization.
const SignBunnyIFrameRegex = `([?&])token=[^&]*&expires=[^&]*`

const (
	BunnyWebhookSignatureVersionHeader   = "X-BunnyStream-Signature-Version"
	BunnyWebhookSignatureAlgorithmHeader = "X-BunnyStream-Signature-Algorithm"
	BunnyWebhookSignatureHeader          = "X-BunnyStream-Signature"
	BunnyWebhookSignatureVersionV1       = "v1"
	BunnyWebhookSignatureAlgorithmHMAC   = "hmac-sha256"
)
