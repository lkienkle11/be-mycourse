package rbac

import (
	"gorm.io/gorm"

	"mycourse-io-be/models"
	"mycourse-io-be/pkg/rbaccatalog"
)

// SeedRBACDefaults ensures catalog permissions (by permission_id) and baseline roles exist.
func SeedRBACDefaults() error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, e := range rbaccatalog.AllPermissionEntries() {
			p := models.Permission{
				PermissionID:   e.PermissionID,
				PermissionName: e.PermissionName,
				Description:    "",
			}
			if err := tx.Where("permission_id = ?", e.PermissionID).FirstOrCreate(&p).Error; err != nil {
				return err
			}
		}
		roles := []models.Role{
			{Name: "sysadmin", Description: "System-wide administration"},
			{Name: "admin", Description: "Business administration"},
			{Name: "instructor", Description: "Manage and teach courses"},
			{Name: "learner", Description: "Consume learning content"},
		}
		for _, role := range roles {
			r := role
			if err := tx.Where("name = ?", r.Name).FirstOrCreate(&r).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
