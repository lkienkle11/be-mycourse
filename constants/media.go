package constants

type FileProvider string

const (
	FileProviderS3    FileProvider = "S3"
	FileProviderGCS   FileProvider = "GCS"
	FileProviderB2    FileProvider = "B2"
	FileProviderR2    FileProvider = "R2"
	FileProviderBunny FileProvider = "Bunny"
	FileProviderLocal FileProvider = "Local"
)

type FileKind string

const (
	FileKindFile  FileKind = "FILE"
	FileKindVideo FileKind = "VIDEO"
)

type FileStatus string

const (
	FileStatusReady   FileStatus = "READY"
	FileStatusDeleted FileStatus = "DELETED"
	FileStatusFailed  FileStatus = "FAILED"
)

// --- Sub 11: image encode concurrency gate ---

// MaxConcurrentImageEncode caps simultaneous WebP encode workers (bimg/libvips) per process.
const MaxConcurrentImageEncode = 4
