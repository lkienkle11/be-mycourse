package media

import "sync/atomic"

var (
	CleanupCloudDeleted atomic.Uint64
	CleanupCloudFailed  atomic.Uint64
	CleanupCloudRetried atomic.Uint64
)
