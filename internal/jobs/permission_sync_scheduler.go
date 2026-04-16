package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/rbacsync"
)

const permissionSyncJobInterval = 12 * time.Hour

var (
	permJobMu     sync.Mutex
	permJobCancel context.CancelFunc
	permJobWG     sync.WaitGroup
)

// StopPermissionSyncJob stops the in-memory permission sync ticker, if any.
func StopPermissionSyncJob() {
	permJobMu.Lock()
	c := permJobCancel
	permJobCancel = nil
	permJobMu.Unlock()
	if c == nil {
		return
	}
	c()
	permJobWG.Wait()
}

// StartPermissionSyncJob starts (or replaces) a background job that syncs permissions from
// constants on a 12h ticker; the first run happens immediately.
func StartPermissionSyncJob(db *gorm.DB) {
	if db == nil {
		log.Println("permission-sync-job: skipped (nil database)")
		return
	}
	StopPermissionSyncJob()

	ctx, cancel := context.WithCancel(context.Background())
	permJobMu.Lock()
	permJobCancel = cancel
	permJobMu.Unlock()

	permJobWG.Add(1)
	go func() {
		defer permJobWG.Done()
		runPermissionSyncLoop(ctx, db)
	}()

	log.Printf("permission-sync-job: started (interval=%s)", permissionSyncJobInterval)
}

func runPermissionSyncLoop(ctx context.Context, db *gorm.DB) {
	runOnce := func() {
		n, err := rbacsync.SyncPermissionsFromConstants(db)
		if err != nil {
			log.Printf("permission-sync-job: failed: %v", err)
			return
		}
		log.Printf("permission-sync-job: synced %d permissions", n)
	}

	runOnce()
	ticker := time.NewTicker(permissionSyncJobInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("permission-sync-job: stopped")
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
