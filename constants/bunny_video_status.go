package constants

// Bunny Stream status values (must match upstream numeric codes).
const (
	BunnyQueued                      = 0
	BunnyProcessing                  = 1
	BunnyEncoding                    = 2
	BunnyFinished                    = 3
	BunnyResolutionFinished          = 4
	BunnyFailed                      = 5
	BunnyPresignedUploadStarted      = 6
	BunnyPresignedUploadFinished     = 7
	BunnyPresignedUploadFailed       = 8
	BunnyCaptionsGenerated           = 9
	BunnyTitleOrDescriptionGenerated = 10
)
