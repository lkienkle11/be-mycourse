package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/rbacsync"
)

const rolePermissionSyncJobInterval = 12 * time.Hour

var (
	rpJobMu     sync.Mutex
	rpJobCancel context.CancelFunc
	rpJobWG     sync.WaitGroup
)

// StopRolePermissionSyncJob stops the in-memory role+permission sync ticker, if any.
func StopRolePermissionSyncJob() {
	rpJobMu.Lock()
	c := rpJobCancel
	rpJobCancel = nil
	rpJobMu.Unlock()
	if c == nil {
		return
	}
	c()
	rpJobWG.Wait()
}

// StartRolePermissionSyncJob starts (or replaces) a background job that rebuilds role_permissions
// from constants on a 12h ticker; the first run happens immediately.
func StartRolePermissionSyncJob(db *gorm.DB) {
	if db == nil {
		log.Println("role-permission-sync-job: skipped (nil database)")
		return
	}
	StopRolePermissionSyncJob()

	ctx, cancel := context.WithCancel(context.Background())
	rpJobMu.Lock()
	rpJobCancel = cancel
	rpJobMu.Unlock()

	rpJobWG.Add(1)
	go func() {
		defer rpJobWG.Done()
		runRolePermissionSyncLoop(ctx, db)
	}()

	log.Printf("role-permission-sync-job: started (interval=%s)", rolePermissionSyncJobInterval)
}

func runRolePermissionSyncLoop(ctx context.Context, db *gorm.DB) {
	runOnce := func() {
		n, err := rbacsync.SyncRolePermissionsFromConstants(db)
		if err != nil {
			log.Printf("role-permission-sync-job: failed: %v", err)
			return
		}
		log.Printf("role-permission-sync-job: rebuilt %d role_permission rows", n)
	}

	runOnce()
	ticker := time.NewTicker(rolePermissionSyncJobInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("role-permission-sync-job: stopped")
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
