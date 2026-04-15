package jobs

import (
	"log"
	"time"

	"gorm.io/gorm"
)

const permissionSyncInterval = 12 * time.Hour

// StartAutoSyncPermissionJob runs a background ticker that syncs permissions every 12 hours.
func StartAutoSyncPermissionJob(db *gorm.DB) {
	if db == nil {
		log.Println("auto-sync-permission: skipped (nil database)")
		return
	}

	runOnce := func() {
		n, err := SyncPermissionsFromConstants(db)
		if err != nil {
			log.Printf("auto-sync-permission: failed: %v", err)
			return
		}
		log.Printf("auto-sync-permission: synced %d permissions", n)
	}

	runOnce()

	ticker := time.NewTicker(permissionSyncInterval)
	go func() {
		for range ticker.C {
			runOnce()
		}
	}()

	log.Printf("auto-sync-permission: enabled (interval=%s)", permissionSyncInterval)
}
