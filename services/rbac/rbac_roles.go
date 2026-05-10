package rbac

import (
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	errfuncdb "mycourse-io-be/pkg/errors_func/db"
	errfuncrbac "mycourse-io-be/pkg/errors_func/rbac"
	"mycourse-io-be/pkg/sqlnamed"
)

// --- Roles ---

func ListRoles(withPerms bool) ([]models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	q := db.Order("name")
	if withPerms {
		q = q.Preload("Permissions")
	}
	var rows []models.Role
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func GetRole(id uint, withPerms bool) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	q := db
	if withPerms {
		q = q.Preload("Permissions")
	}
	var r models.Role
	if err := q.First(&r, id).Error; err != nil {
		return nil, errfuncdb.MapRecordNotFound(err)
	}
	return &r, nil
}

func CreateRole(name, description string) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, pkgerrors.ErrRBACRoleNameRequired
	}
	out := models.Role{Name: name, Description: description}
	if err := db.Create(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func UpdateRole(id uint, name *string, description *string) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	var r models.Role
	if err := db.First(&r, id).Error; err != nil {
		return nil, errfuncdb.MapRecordNotFound(err)
	}
	if name != nil && *name != "" {
		r.Name = *name
	}
	if description != nil {
		r.Description = *description
	}
	if err := db.Save(&r).Error; err != nil {
		return nil, err
	}
	return GetRole(id, true)
}

func DeleteRole(id uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		q, args, err := sqlnamed.Postgres(rbacSQLDeleteRolePermissionsByRoleID, map[string]interface{}{"role_id": id})
		if err != nil {
			return err
		}
		if err := tx.Exec(q, args...).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Role{}, id).Error
	})
}

func replaceRolePermissionRows(tx *gorm.DB, roleID uint, permissionIDs []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		var n int64
		if err := tx.Model(&models.Permission{}).Where("permission_id = ?", pid).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return errfuncrbac.WrapRBACUnknownPermissionID(pid)
		}
		if err := tx.Create(&models.RolePermission{RoleID: roleID, PermissionID: pid}).Error; err != nil {
			return err
		}
	}
	return nil
}

// SetRolePermissions replaces all permissions on the role using permission_id values (e.g. P1).
func SetRolePermissions(roleID uint, permissionIDs []string) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	var role models.Role
	if err := db.First(&role, roleID).Error; err != nil {
		return nil, errfuncdb.MapRecordNotFound(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return replaceRolePermissionRows(tx, roleID, permissionIDs)
	}); err != nil {
		return nil, err
	}
	return GetRole(roleID, true)
}
