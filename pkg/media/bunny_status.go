package media

import "mycourse-io-be/constants"

var bunnyStatusNames = map[int]string{
	constants.BunnyQueued:                      "queued",
	constants.BunnyProcessing:                  "processing",
	constants.BunnyEncoding:                    "encoding",
	constants.BunnyFinished:                    "finished",
	constants.BunnyResolutionFinished:          "resolution_finished",
	constants.BunnyFailed:                      "failed",
	constants.BunnyPresignedUploadStarted:      "presigned_upload_started",
	constants.BunnyPresignedUploadFinished:     "presigned_upload_finished",
	constants.BunnyPresignedUploadFailed:       "presigned_upload_failed",
	constants.BunnyCaptionsGenerated:           "captions_generated",
	constants.BunnyTitleOrDescriptionGenerated: "title_or_description_generated",
}

func BunnyStatusString(status int) string {
	if name, ok := bunnyStatusNames[status]; ok {
		return name
	}
	return "unknown"
}
