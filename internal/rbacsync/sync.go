package rbacsync

import (
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
)

// SyncPermissionsFromConstants upserts rows from constants.AllPermissionEntries by permission_id:
// updates permission_name (and leaves any extra DB permissions untouched).
func SyncPermissionsFromConstants(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, errors.New("nil database")
	}

	entries := constants.AllPermissionEntries()
	if len(entries) == 0 {
		return 0, errors.New("no permission fields tagged with perm_id in constants.AllPermissions")
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			if err := upsertPermissionCatalogRow(tx, e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return len(entries), nil
}

func upsertPermissionCatalogRow(tx *gorm.DB, e constants.PermissionCatalogEntry) error {
	var p models.Permission
	err := tx.Where("permission_id = ?", e.PermissionID).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		p = models.Permission{
			PermissionID:   e.PermissionID,
			PermissionName: e.PermissionName,
			Description:    "",
		}
		return tx.Create(&p).Error
	}
	if err != nil {
		return err
	}
	p.PermissionName = e.PermissionName
	return tx.Save(&p).Error
}
