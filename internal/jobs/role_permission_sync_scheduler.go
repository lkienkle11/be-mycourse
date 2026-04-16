package jobs

import (
	"log"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/rbacsync"
)

const rolePermissionSyncInterval = 7 * 24 * time.Hour

// StartWeeklyRolePermissionSyncJob runs a background ticker that rebuilds role_permissions
// from constants.RolePermissions (same as cmd/syncrolepermissions). First run is immediate.
func StartWeeklyRolePermissionSyncJob(db *gorm.DB) {
	if db == nil {
		log.Println("auto-sync-role-permission: skipped (nil database)")
		return
	}

	runOnce := func() {
		n, err := rbacsync.SyncRolePermissionsFromConstants(db)
		if err != nil {
			log.Printf("auto-sync-role-permission: failed: %v", err)
			return
		}
		log.Printf("auto-sync-role-permission: rebuilt %d role_permission rows", n)
	}

	runOnce()

	ticker := time.NewTicker(rolePermissionSyncInterval)
	go func() {
		for range ticker.C {
			runOnce()
		}
	}()

	log.Printf("auto-sync-role-permission: enabled (interval=%s)", rolePermissionSyncInterval)
}
