// Command syncrolepermissions replaces all role_permissions rows from constants.RolePermissions
// (roles resolved by name; permission_id taken from struct tags). Run after migrations and
// optionally after syncpermissions, e.g.:
//
//	go run ./cmd/syncrolepermissions
package main

import (
	"log"

	"mycourse-io-be/internal/rbacsync"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/setting"
)

func main() {
	log.SetFlags(0)
	if err := setting.Setup(); err != nil {
		log.Fatalf("syncrolepermissions: setup setting: %v", err)
	}
	if err := models.Setup(); err != nil {
		log.Fatalf("syncrolepermissions: setup postgres: %v", err)
	}

	n, err := rbacsync.SyncRolePermissionsFromConstants(models.DB)
	if err != nil {
		log.Fatalf("syncrolepermissions: %v", err)
	}

	log.Printf("syncrolepermissions: ok (%d role_permission rows)", n)
}
