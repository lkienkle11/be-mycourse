package rbacsync

import (
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
)

// SyncPermissionsFromConstants upserts permissions from constants and prunes unknown DB rows.
func SyncPermissionsFromConstants(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, errors.New("nil database")
	}

	entries := constants.AllPermissionEntries()
	if len(entries) == 0 {
		return 0, errors.New("no permission fields tagged with perm in constants")
	}

	codes := make([]string, 0, len(entries))
	for _, e := range entries {
		codes = append(codes, e.Code)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			var p models.Permission
			err := tx.Where("code = ?", e.Code).First(&p).Error
			if err == gorm.ErrRecordNotFound {
				p = models.Permission{
					Code:        e.Code,
					Action:      e.Action,
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
			p.Action = e.Action
			if err := tx.Save(&p).Error; err != nil {
				return err
			}
		}
		return tx.Where("code NOT IN ?", codes).Delete(&models.Permission{}).Error
	})
	if err != nil {
		return 0, err
	}

	return len(entries), nil
}
