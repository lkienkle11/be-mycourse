// Command syncpermissions upserts permissions.permission_name from constants.AllPermissions
// for each perm_id tag (extra DB rows are left unchanged). Run before release builds, e.g.:
//
//	go run ./cmd/syncpermissions
package main

import (
	"context"
	"log"

	"mycourse-io-be/internal/shared/db"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/system/application"
	"mycourse-io-be/internal/system/infra"
)

func main() {
	log.SetFlags(0)
	if err := setting.Setup(); err != nil {
		log.Fatalf("syncpermissions: setup setting: %v", err)
	}
	if err := db.Setup(); err != nil {
		log.Fatalf("syncpermissions: setup postgres: %v", err)
	}

	syncer := infra.NewGormPermissionSyncer(db.Conn())
	entries := application.AllPermissionEntries()
	count, err := syncer.SyncPermissionsFromCatalog(context.Background(), entries)
	if err != nil {
		log.Fatalf("syncpermissions: %v", err)
	}

	log.Printf("syncpermissions: ok (%d catalog entries synced by permission_id)", count)
}
