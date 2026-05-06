package constants

// Bunny Stream literals only (Global Constants Placement). Status enum + StatusString() live in bunny_video_status.go.

// FinishedWebhookBunnyStatus is the Bunny webhook numeric status meaning processing finished (skeleton sync).
const FinishedWebhookBunnyStatus = int(BunnyFinished)

// SignBunnyIFrameRegex matches signed iframe query noise for cleanup/normalization.
const SignBunnyIFrameRegex = `([?&])token=[^&]*&expires=[^&]*`
