package application

// CreateFileInput carries request parameters for single/batch create.
type CreateFileInput struct {
	Kind      string
	ObjectKey string
	Metadata  map[string]any
}

// UpdateFileInput carries request parameters for single/bundle update.
type UpdateFileInput struct {
	Kind                  string
	Metadata              map[string]any
	ReuseMediaID          string
	ExpectedRowVersion    *int64
	SkipUploadIfUnchanged bool
}

// BunnyWebhookInput carries the decoded Bunny Stream webhook payload.
type BunnyWebhookInput struct {
	VideoGUID string
	Status    int
}
