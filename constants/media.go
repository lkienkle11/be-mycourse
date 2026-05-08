package constants

const (
	FileProviderS3    = "S3"
	FileProviderGCS   = "GCS"
	FileProviderB2    = "B2"
	FileProviderR2    = "R2"
	FileProviderBunny = "Bunny"
	FileProviderLocal = "Local"
	FileKindFile      = "FILE"
	FileKindVideo     = "VIDEO"
	FileStatusReady   = "READY"
	FileStatusDeleted = "DELETED"
	FileStatusFailed  = "FAILED"
)

// --- Sub 11: image encode concurrency gate ---

// MaxConcurrentImageEncode caps simultaneous WebP encode workers (bimg/libvips) per process.
const MaxConcurrentImageEncode = 4
