package constants

// BunnyVideoStatus is Bunny Stream’s numeric video status from the API.
type BunnyVideoStatus int

// Bunny Stream status values (must match upstream numeric codes).
const (
	BunnyCreated             BunnyVideoStatus = 0
	BunnyUploaded            BunnyVideoStatus = 1
	BunnyProcessing          BunnyVideoStatus = 2
	BunnyTranscoding         BunnyVideoStatus = 3
	BunnyFinished            BunnyVideoStatus = 4
	BunnyResolutionsFinished BunnyVideoStatus = 5
	BunnyFailed              BunnyVideoStatus = 6
	BunnyPresignedUpload     BunnyVideoStatus = 7
	BunnyTranscribing        BunnyVideoStatus = 8
)

// StatusString maps a numeric Bunny status to a stable API string (unknown fallback).
func (s BunnyVideoStatus) StatusString() string {
	switch s {
	case BunnyCreated:
		return "created"
	case BunnyUploaded:
		return "uploaded"
	case BunnyProcessing:
		return "processing"
	case BunnyTranscoding:
		return "transcoding"
	case BunnyFinished:
		return "finished"
	case BunnyResolutionsFinished:
		return "resolutions_finished"
	case BunnyFailed:
		return "failed"
	case BunnyPresignedUpload:
		return "presigned_upload"
	case BunnyTranscribing:
		return "transcribing"
	default:
		return "unknown"
	}
}
