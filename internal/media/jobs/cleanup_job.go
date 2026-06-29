// Package jobs contains background workers for the MEDIA bounded context.
package jobs

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
)

const (
	MediaCleanupDefaultIntervalSec = 300
	MediaCleanupBatchSize          = 50
	MediaCleanupMaxAttempts        = 5
)

var (
	CleanupCloudDeleted atomic.Uint64
	CleanupCloudFailed  atomic.Uint64
	CleanupCloudRetried atomic.Uint64

	GlobalCounters = &counterAdaptor{}

	mediaCleanupMu     sync.Mutex
	mediaCleanupCancel context.CancelFunc
	mediaCleanupWG     sync.WaitGroup
)

type counterAdaptor struct{}

func (c *counterAdaptor) Deleted() uint64 { return CleanupCloudDeleted.Load() }
func (c *counterAdaptor) Failed() uint64  { return CleanupCloudFailed.Load() }
func (c *counterAdaptor) Retried() uint64 { return CleanupCloudRetried.Load() }

// ProcessPendingCleanupBatch executes one batch of deferred cloud deletes from media_pending_cloud_cleanup.
func ProcessPendingCleanupBatch(ctx context.Context, cleanupRepo domain.PendingCleanupRepository) {
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return
	}
	rows, err := cleanupRepo.FindPending(ctx, MediaCleanupBatchSize)
	if err != nil || len(rows) == 0 {
		return
	}
	clients := mediainfra.Cloud
	for _, row := range rows {
		delErr := mediainfra.DeleteStoredObject(ctx, clients, row.ObjectKey, row.Provider, row.BunnyVideoID)
		next := row.AttemptCount + 1
		if delErr == nil {
			_ = cleanupRepo.MarkDone(ctx, row.ID)
			CleanupCloudDeleted.Add(1)
			continue
		}
		if next >= MediaCleanupMaxAttempts {
			_ = cleanupRepo.MarkFailed(ctx, row.ID, delErr.Error(), nil)
			CleanupCloudFailed.Add(1)
			continue
		}
		backoff := time.Duration(next*next) * time.Minute
		if backoff > 30*time.Minute {
			backoff = 30 * time.Minute
		}
		nextRunAt := time.Now().Add(backoff)
		_ = cleanupRepo.MarkFailed(ctx, row.ID, delErr.Error(), nextRunAt)
		CleanupCloudRetried.Add(1)
	}
}

func mediaPendingCleanupIntervalFromEnv() time.Duration {
	s := strings.TrimSpace(os.Getenv("MEDIA_CLEANUP_INTERVAL_SEC"))
	if s == "0" {
		return 0
	}
	if s == "" {
		return time.Duration(MediaCleanupDefaultIntervalSec) * time.Second
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec <= 0 {
		return time.Duration(MediaCleanupDefaultIntervalSec) * time.Second
	}
	return time.Duration(sec) * time.Second
}

// StopMediaPendingCleanupJob stops the background pending-cleanup worker.
func StopMediaPendingCleanupJob() {
	mediaCleanupMu.Lock()
	c := mediaCleanupCancel
	mediaCleanupCancel = nil
	mediaCleanupMu.Unlock()
	if c == nil {
		return
	}
	c()
	mediaCleanupWG.Wait()
}

// StartMediaPendingCleanupJob starts (or replaces) the media pending cloud cleanup loop.
func StartMediaPendingCleanupJob(cleanupRepo domain.PendingCleanupRepository) {
	if cleanupRepo == nil {
		zap.L().Info("media pending cleanup job skipped", zap.String("reason", "nil repository"))
		return
	}
	interval := mediaPendingCleanupIntervalFromEnv()
	if interval <= 0 {
		zap.L().Info("media pending cleanup job skipped", zap.String("reason", "MEDIA_CLEANUP_INTERVAL_SEC=0"))
		return
	}
	StopMediaPendingCleanupJob()

	ctx, cancel := context.WithCancel(context.Background())
	mediaCleanupMu.Lock()
	mediaCleanupCancel = cancel
	mediaCleanupMu.Unlock()

	mediaCleanupWG.Add(1)
	go func() {
		defer mediaCleanupWG.Done()
		runMediaPendingCleanupLoop(ctx, cleanupRepo, interval)
	}()

	zap.L().Info("media pending cleanup job started", zap.Duration("interval", interval))
}

func runMediaPendingCleanupLoop(ctx context.Context, cleanupRepo domain.PendingCleanupRepository, interval time.Duration) {
	runOnce := func() { ProcessPendingCleanupBatch(context.Background(), cleanupRepo) }

	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("media pending cleanup job stopped")
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
