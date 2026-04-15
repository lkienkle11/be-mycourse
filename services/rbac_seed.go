package services

import (
	"gorm.io/gorm"

	"mycourse-io-be/models"
)

// defaultPermissionSeeds are created on seed if missing.
var defaultPermissionSeeds = []struct {
	Code        string
	Action      string
	Description string
}{
	{"user.read", "user:read", ""},
	{"user.create", "user:create", ""},
	{"user.update", "user:update", ""},
	{"user.delete", "user:delete", ""},
	{"course.read", "course:read", ""},
	{"course.create", "course:create", ""},
	{"course.update", "course:update", ""},
	{"course.delete", "course:delete", ""},
	{"user_admin.read", "user_admin:read", ""},
	{"user_admin.create", "user_admin:create", ""},
	{"user_admin.update", "user_admin:update", ""},
	{"user_admin.delete", "user_admin:delete", ""},
}

// SeedRBACDefaults creates baseline permissions and all default roles.
func SeedRBACDefaults() error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, s := range defaultPermissionSeeds {
			p := models.Permission{Code: s.Code, Action: s.Action, Description: s.Description}
			if err := tx.Where("code = ?", s.Code).FirstOrCreate(&p).Error; err != nil {
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
