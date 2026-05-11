package jobrbac

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
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
		zap.L().Info("rbac sync job skipped", zap.String("job", logLabel), zap.String("reason", "nil database"))
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

	zap.L().Info("rbac sync job started", zap.String("job", logLabel), zap.Duration("interval", b.interval))
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
			zap.L().Error("permission sync job tick failed", zap.Error(err))
			return
		}
		zap.L().Info("permission sync job synced", zap.Int("count", n))
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
			zap.L().Error("role-permission sync job tick failed", zap.Error(err))
			return
		}
		zap.L().Info("role-permission sync job rebuilt", zap.Int("rows", n))
	})
}
