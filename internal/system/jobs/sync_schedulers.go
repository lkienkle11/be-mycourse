// Package jobs contains background workers for the SYSTEM bounded context.
package jobs

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"mycourse-io-be/internal/system/application"
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

func (b *syncJobBundle) start(svc *application.SystemService, logLabel string, onTick func(*application.SystemService)) {
	if svc == nil {
		zap.L().Info("rbac sync job skipped", zap.String("job", logLabel), zap.String("reason", "nil service"))
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
			onTick(svc)
		})
	}()

	zap.L().Info("rbac sync job started", zap.String("job", logLabel), zap.Duration("interval", b.interval))
}

var (
	permissionSyncJob     = syncJobBundle{interval: rbacBackgroundSyncInterval}
	rolePermissionSyncJob = syncJobBundle{interval: rbacBackgroundSyncInterval}
)

// StopPermissionSyncJob stops the in-memory permission sync ticker.
func StopPermissionSyncJob() { permissionSyncJob.stop() }

// StartPermissionSyncJob starts (or replaces) a background job that syncs permissions from
// constants on a 12h ticker; the first run happens immediately.
func StartPermissionSyncJob(svc *application.SystemService) {
	permissionSyncJob.start(svc, "permission-sync-job", func(svc *application.SystemService) {
		n, err := svc.SyncPermissions(context.Background())
		if err != nil {
			zap.L().Error("permission sync job tick failed", zap.Error(err))
			return
		}
		zap.L().Info("permission sync job synced", zap.Int("count", n))
	})
}

// StopRolePermissionSyncJob stops the in-memory role+permission sync ticker.
func StopRolePermissionSyncJob() { rolePermissionSyncJob.stop() }

// StartRolePermissionSyncJob starts (or replaces) a background job that rebuilds role_permissions
// from constants on a 12h ticker; the first run happens immediately.
func StartRolePermissionSyncJob(svc *application.SystemService) {
	rolePermissionSyncJob.start(svc, "role-permission-sync-job", func(svc *application.SystemService) {
		n, err := svc.SyncRolePermissions(context.Background())
		if err != nil {
			zap.L().Error("role-permission sync job tick failed", zap.Error(err))
			return
		}
		zap.L().Info("role-permission sync job rebuilt", zap.Int("rows", n))
	})
}

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
