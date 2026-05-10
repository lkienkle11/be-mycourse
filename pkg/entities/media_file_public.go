package entities

// MediaFilePublic is the client-facing subset of a stored media row (no origin_url / server-only fields).
// dto.MediaFilePublic is a type alias to this type so JSON shape and domain/cache payloads stay single-sourced.
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
