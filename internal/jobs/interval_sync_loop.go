package jobs

import (
	"context"
	"log"
	"time"
)

// runPeriodicSyncLoop runs runOnce immediately, then on every tick until ctx is cancelled.
func runPeriodicSyncLoop(ctx context.Context, interval time.Duration, jobName string, runOnce func()) {
	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("%s: stopped", jobName)
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
