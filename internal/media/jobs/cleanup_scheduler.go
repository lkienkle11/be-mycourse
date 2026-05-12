package jobs

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"mycourse-io-be/internal/media/domain" //nolint:depguard // jobs use domain.PendingCleanupRepository interface as parameter type
)

var (
	mediaCleanupMu     sync.Mutex
	mediaCleanupCancel context.CancelFunc
	mediaCleanupWG     sync.WaitGroup
)

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
