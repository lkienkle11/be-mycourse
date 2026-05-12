// Command syncrolepermissions replaces all role_permissions rows from constants.RolePermissions
// (roles resolved by name; permission_id taken from struct tags). Run after migrations and
// optionally after syncpermissions, e.g.:
//
//	go run ./cmd/syncrolepermissions
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
		log.Fatalf("syncrolepermissions: setup setting: %v", err)
	}
	if err := db.Setup(); err != nil {
		log.Fatalf("syncrolepermissions: setup postgres: %v", err)
	}

	syncer := infra.NewGormRolePermissionSyncer(db.Conn())
	pairs := application.AllRolePermissionPairs()
	n, err := syncer.SyncRolePermissionsFromCatalog(context.Background(), pairs)
	if err != nil {
		log.Fatalf("syncrolepermissions: %v", err)
	}

	log.Printf("syncrolepermissions: ok (%d role_permission rows)", n)
}
