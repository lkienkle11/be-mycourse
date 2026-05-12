package infra

import "mycourse-io-be/internal/media/domain"

var bunnyStatusNames = map[int]string{
	domain.BunnyQueued:                      "queued",
	domain.BunnyProcessing:                  "processing",
	domain.BunnyEncoding:                    "encoding",
	domain.BunnyFinished:                    "finished",
	domain.BunnyResolutionFinished:          "resolution_finished",
	domain.BunnyFailed:                      "failed",
	domain.BunnyPresignedUploadStarted:      "presigned_upload_started",
	domain.BunnyPresignedUploadFinished:     "presigned_upload_finished",
	domain.BunnyPresignedUploadFailed:       "presigned_upload_failed",
	domain.BunnyCaptionsGenerated:           "captions_generated",
	domain.BunnyTitleOrDescriptionGenerated: "title_or_description_generated",
}

func BunnyStatusString(status int) string {
	if name, ok := bunnyStatusNames[status]; ok {
		return name
	}
	return "unknown"
}
