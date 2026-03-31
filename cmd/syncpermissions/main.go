// Command syncpermissions aligns the permissions table with all `perm:"..."` fields in constants (upsert by Code,
// update CodeCheck, delete rows whose Code is not in the catalog). Run before release builds, e.g.:
//
//	go run ./cmd/syncpermissions
package main

import (
	"log"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
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

	entries := constants.AllPermissionEntries()
	if len(entries) == 0 {
		log.Fatal("syncpermissions: no permission fields tagged with perm in constants")
	}

	codes := make([]string, 0, len(entries))
	for _, e := range entries {
		codes = append(codes, e.Code)
	}

	err := models.DB.Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			var p models.Permission
			err := tx.Where("code = ?", e.Code).First(&p).Error
			if err == gorm.ErrRecordNotFound {
				p = models.Permission{
					Code:        e.Code,
					CodeCheck:   e.CodeCheck,
					Description: "",
				}
				if err := tx.Create(&p).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			p.CodeCheck = e.CodeCheck
			if err := tx.Save(&p).Error; err != nil {
				return err
			}
		}
		return tx.Where("code NOT IN ?", codes).Delete(&models.Permission{}).Error
	})
	if err != nil {
		log.Fatalf("syncpermissions: %v", err)
	}

	log.Printf("syncpermissions: ok (%d catalog entries; DB pruned to same set of codes)", len(entries))
}
