// Command syncpermissions aligns the permissions table with all `perm:"..."` fields in constants (upsert by Code,
// update Action, delete rows whose Code is not in the catalog). Run before release builds, e.g.:
//
//	go run ./cmd/syncpermissions
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
		log.Fatalf("syncpermissions: setup setting: %v", err)
	}
	if err := models.Setup(); err != nil {
		log.Fatalf("syncpermissions: setup postgres: %v", err)
	}

	count, err := rbacsync.SyncPermissionsFromConstants(models.DB)
	if err != nil {
		log.Fatalf("syncpermissions: %v", err)
	}

	log.Printf("syncpermissions: ok (%d catalog entries; DB pruned to same set of codes)", count)
}
