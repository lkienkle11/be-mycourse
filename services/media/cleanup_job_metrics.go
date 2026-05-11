package media

import jobmedia "mycourse-io-be/internal/jobs/media"

// PendingCloudCleanupCounters returns atomic counters from the media pending-cleanup job (services may call internal/jobs per Rule 6).
func PendingCloudCleanupCounters() (deleted, failed, retried uint64) {
	return jobmedia.CleanupCloudDeleted.Load(), jobmedia.CleanupCloudFailed.Load(), jobmedia.CleanupCloudRetried.Load()
}
