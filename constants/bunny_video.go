package constants

// Bunny Stream literals only (Global Constants Placement). Status enum + mapper live in pkg/logic/helper.

// FinishedWebhookBunnyStatus is the Bunny webhook numeric status meaning processing finished (skeleton sync).
const FinishedWebhookBunnyStatus = 4

// SignBunnyIFrameRegex matches signed iframe query noise for cleanup/normalization.
const SignBunnyIFrameRegex = `([?&])token=[^&]*&expires=[^&]*`
