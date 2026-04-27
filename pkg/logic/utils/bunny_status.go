package utils

const FinishedWebhookBunnyStatus = 4

type BunnyVideoStatus int

const (
	Created             BunnyVideoStatus = 0
	Uploaded            BunnyVideoStatus = 1
	Processing          BunnyVideoStatus = 2
	Transcoding         BunnyVideoStatus = 3
	Finished            BunnyVideoStatus = 4
	ResolutionsFinished BunnyVideoStatus = 5
	Failed              BunnyVideoStatus = 6
	PresignedUpload     BunnyVideoStatus = 7
	Transcribing        BunnyVideoStatus = 8
)

func (s BunnyVideoStatus) StatusString() string {
	switch s {
	case Created:
		return "created"
	case Uploaded:
		return "uploaded"
	case Processing:
		return "processing"
	case Transcoding:
		return "transcoding"
	case Finished:
		return "finished"
	case ResolutionsFinished:
		return "resolutions_finished"
	case Failed:
		return "failed"
	case PresignedUpload:
		return "presigned_upload"
	case Transcribing:
		return "transcribing"
	default:
		return "unknown"
	}
}
