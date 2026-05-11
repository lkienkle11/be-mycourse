package jobrbac

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// runPeriodicSyncLoop runs runOnce immediately, then on every tick until ctx is cancelled.
func runPeriodicSyncLoop(ctx context.Context, interval time.Duration, jobName string, runOnce func()) {
	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("rbac periodic job stopped", zap.String("job", jobName))
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
