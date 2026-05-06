package dto

// MediaFilePublic is the client-facing subset of a stored media row (aligned with
// pkg/entities.File public fields; omits origin_url and other server-only values).
type MediaFilePublic struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Provider           string `json:"provider"`
	Filename           string `json:"filename"`
	MimeType           string `json:"mime_type"`
	SizeBytes          int64  `json:"size_bytes"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	URL                string `json:"url"`
	Duration           int64  `json:"duration"`
	ContentFingerprint string `json:"content_fingerprint"`
	Status             string `json:"status"`
}
