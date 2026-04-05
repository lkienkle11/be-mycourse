package services

import (
	"gorm.io/gorm"

	"mycourse-io-be/models"
)

// defaultPermissionSeeds are created on seed if missing.
var defaultPermissionSeeds = []struct {
	Code        string
	CodeCheck   string
	Description string
}{
	{"rbac.manage", "rbac:manage", "Manage roles, permissions, and user-role assignments"},
	{"profile.course.read", "profile:course:read", "Read own profile"},
}

// SeedRBACDefaults creates baseline permissions and an admin role with rbac.manage.
func SeedRBACDefaults() error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, s := range defaultPermissionSeeds {
			p := models.Permission{Code: s.Code, CodeCheck: s.CodeCheck, Description: s.Description}
			if err := tx.Where("code = ?", s.Code).FirstOrCreate(&p).Error; err != nil {
				return err
			}
		}
		var manage models.Permission
		if err := tx.Where("code = ?", "rbac.manage").First(&manage).Error; err != nil {
			return err
		}
		admin := models.Role{Name: "admin", Description: "Full RBAC administration"}
		if err := tx.Where("name = ?", "admin").FirstOrCreate(&admin).Error; err != nil {
			return err
		}
		return tx.Model(&admin).Association("Permissions").Replace([]models.Permission{manage})
	})
}
