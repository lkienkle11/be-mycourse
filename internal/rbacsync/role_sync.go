package rbacsync

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/dbschema"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/rbaccatalog"
)

// SyncRolePermissionsFromConstants replaces all role_permissions rows with the matrix from
// constants.AllRolePermissionPairs. Role rows are resolved by name; permission_id is taken
// verbatim from constants (no lookup by permission_name).
func SyncRolePermissionsFromConstants(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, errors.New("nil database")
	}
	pairs := rbaccatalog.AllRolePermissionPairs()
	if len(pairs) == 0 {
		return 0, errors.New("no role-permission pairs in constants.RolePermissions")
	}
	tbl := dbschema.RBAC.RolePermissions()
	inserted := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", tbl)).Error; err != nil {
			return err
		}
		for _, pair := range pairs {
			var role models.Role
			if err := tx.Where("name = ?", pair.RoleName).First(&role).Error; err != nil {
				return fmt.Errorf("role %q: %w", pair.RoleName, err)
			}
			row := models.RolePermission{RoleID: role.ID, PermissionID: pair.PermID}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("role %q perm %q: %w", pair.RoleName, pair.PermID, err)
			}
			inserted++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}
