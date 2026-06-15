package jobs

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	mediaapp "mycourse-io-be/internal/media/application"
)

const videoDurationBackfillStartupDelay = 2 * time.Minute

var (
	videoDurationBackfillMu     sync.Mutex
	videoDurationBackfillCancel context.CancelFunc
	videoDurationBackfillWG     sync.WaitGroup
)

// StartVideoDurationBackfillJob runs a one-shot backfill after a startup delay so
// Bunny/DB work does not race HTTP readiness or trip the circuit breaker.
func StartVideoDurationBackfillJob(svc *mediaapp.MediaService) {
	if svc == nil {
		return
	}
	videoDurationBackfillMu.Lock()
	defer videoDurationBackfillMu.Unlock()
	if videoDurationBackfillCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	videoDurationBackfillCancel = cancel
	videoDurationBackfillWG.Add(1)
	go func() {
		defer videoDurationBackfillWG.Done()
		timer := time.NewTimer(videoDurationBackfillStartupDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			zap.L().Info("media video duration backfill starting")
			svc.BackfillMissingVideoDurations(ctx)
			zap.L().Info("media video duration backfill finished")
		}
	}()
}

// StopVideoDurationBackfillJob cancels a pending startup backfill.
func StopVideoDurationBackfillJob() {
	videoDurationBackfillMu.Lock()
	cancel := videoDurationBackfillCancel
	videoDurationBackfillCancel = nil
	videoDurationBackfillMu.Unlock()
	if cancel != nil {
		cancel()
	}
	videoDurationBackfillWG.Wait()
}
