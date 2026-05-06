package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/rbacsync"
)

const rbacBackgroundSyncInterval = 12 * time.Hour

type syncJobBundle struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	interval time.Duration
}

func (b *syncJobBundle) stop() {
	b.mu.Lock()
	c := b.cancel
	b.cancel = nil
	b.mu.Unlock()
	if c == nil {
		return
	}
	c()
	b.wg.Wait()
}

func (b *syncJobBundle) start(db *gorm.DB, logLabel string, onTick func(*gorm.DB)) {
	if db == nil {
		log.Printf("%s: skipped (nil database)", logLabel)
		return
	}
	b.stop()

	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.cancel = cancel
	b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		runPeriodicSyncLoop(ctx, b.interval, logLabel, func() {
			onTick(db)
		})
	}()

	log.Printf("%s: started (interval=%s)", logLabel, b.interval)
}

var (
	permissionSyncJob     = syncJobBundle{interval: rbacBackgroundSyncInterval}
	rolePermissionSyncJob = syncJobBundle{interval: rbacBackgroundSyncInterval}
)

// StopPermissionSyncJob stops the in-memory permission sync ticker, if any.
func StopPermissionSyncJob() {
	permissionSyncJob.stop()
}

// StartPermissionSyncJob starts (or replaces) a background job that syncs permissions from
// constants on a 12h ticker; the first run happens immediately.
func StartPermissionSyncJob(db *gorm.DB) {
	permissionSyncJob.start(db, "permission-sync-job", func(db *gorm.DB) {
		n, err := rbacsync.SyncPermissionsFromConstants(db)
		if err != nil {
			log.Printf("permission-sync-job: failed: %v", err)
			return
		}
		log.Printf("permission-sync-job: synced %d permissions", n)
	})
}

// StopRolePermissionSyncJob stops the in-memory role+permission sync ticker, if any.
func StopRolePermissionSyncJob() {
	rolePermissionSyncJob.stop()
}

// StartRolePermissionSyncJob starts (or replaces) a background job that rebuilds role_permissions
// from constants on a 12h ticker; the first run happens immediately.
func StartRolePermissionSyncJob(db *gorm.DB) {
	rolePermissionSyncJob.start(db, "role-permission-sync-job", func(db *gorm.DB) {
		n, err := rbacsync.SyncRolePermissionsFromConstants(db)
		if err != nil {
			log.Printf("role-permission-sync-job: failed: %v", err)
			return
		}
		log.Printf("role-permission-sync-job: rebuilt %d role_permission rows", n)
	})
}
