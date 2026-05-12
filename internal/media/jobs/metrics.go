// Package jobs contains background workers for the MEDIA bounded context.
package jobs

import "sync/atomic"

// CleanupCloudDeleted, CleanupCloudFailed, and CleanupCloudRetried are atomic counters
// exposed by GET /media/files/cleanup-metrics.
var (
	CleanupCloudDeleted atomic.Uint64
	CleanupCloudFailed  atomic.Uint64
	CleanupCloudRetried atomic.Uint64
)

// GlobalCounters is a singleton adaptor that satisfies application.CleanupCounters
// using the three package-level atomic counters.
var GlobalCounters = &counterAdaptor{}

type counterAdaptor struct{}

func (c *counterAdaptor) Deleted() uint64 { return CleanupCloudDeleted.Load() }
func (c *counterAdaptor) Failed() uint64  { return CleanupCloudFailed.Load() }
func (c *counterAdaptor) Retried() uint64 { return CleanupCloudRetried.Load() }
