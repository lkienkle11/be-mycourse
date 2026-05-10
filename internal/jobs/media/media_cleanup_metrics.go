package jobmedia

import "sync/atomic"

// CleanupCloudDeleted / CleanupCloudFailed / CleanupCloudRetried are atomic counters for GET /media/files/cleanup-metrics.
var (
	CleanupCloudDeleted atomic.Uint64
	CleanupCloudFailed  atomic.Uint64
	CleanupCloudRetried atomic.Uint64
)
