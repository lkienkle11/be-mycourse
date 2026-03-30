package services

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/dbschema"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/sqlnamed"
)

// Native RBAC SQL (named value params :name via sqlnamed.Postgres; table names from dbschema in init).
const (
	rbacSQLPermissionCodesForUserTmpl = `
SELECT DISTINCT p.code
FROM %s ur
INNER JOIN %s rc ON rc.descendant_id = ur.role_id
INNER JOIN %s rp ON rp.role_id = rc.ancestor_id
INNER JOIN %s p ON p.id = rp.permission_id
WHERE ur.user_id = :user_id
`
	rbacSQLDeleteRolePermissionsByPermissionIDTmpl = `
DELETE FROM %s WHERE permission_id = :permission_id
`
	rbacSQLDeleteRolePermissionsByRoleIDTmpl = `
DELETE FROM %s WHERE role_id = :role_id
`
)

var (
	rbacSQLPermissionCodesForUser              string
	rbacSQLDeleteRolePermissionsByPermissionID string
	rbacSQLDeleteRolePermissionsByRoleID       string
)

func init() {
	rbacSQLPermissionCodesForUser = fmt.Sprintf(rbacSQLPermissionCodesForUserTmpl,
		dbschema.RBAC.UserRoles(),
		dbschema.RBAC.RoleClosure(),
		dbschema.RBAC.RolePermissions(),
		dbschema.RBAC.Permissions(),
	)
	rbacSQLDeleteRolePermissionsByPermissionID = fmt.Sprintf(rbacSQLDeleteRolePermissionsByPermissionIDTmpl, dbschema.RBAC.RolePermissions())
	rbacSQLDeleteRolePermissionsByRoleID = fmt.Sprintf(rbacSQLDeleteRolePermissionsByRoleIDTmpl, dbschema.RBAC.RolePermissions())
}

var rbacDB *gorm.DB

func SetRBACDB(db *gorm.DB) {
	rbacDB = db
}

func rbacOrErr() (*gorm.DB, error) {
	if rbacDB == nil {
		return nil, errors.New("rbac database not configured")
	}
	return rbacDB, nil
}

// PermissionCodesForUser returns distinct permission codes for the user.
// Uses role_closure so each assigned role inherits permissions from all ancestors in one flat SQL query:
// no recursion, no stack in application code; work is O(1) in tree depth (fixed join count, single round-trip).
func PermissionCodesForUser(userID string) (map[string]struct{}, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, errors.New("empty user id")
	}
	var codes []string
	q, args, err := sqlnamed.Postgres(rbacSQLPermissionCodesForUser, map[string]interface{}{"user_id": userID})
	if err != nil {
		return nil, err
	}
	if err := db.Raw(q, args...).Scan(&codes).Error; err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		out[c] = struct{}{}
	}
	return out, nil
}

func UserHasAllPermissions(userID string, required []string) (bool, string, error) {
	if len(required) == 0 {
		return true, "", nil
	}
	set, err := PermissionCodesForUser(userID)
	if err != nil {
		return false, "", err
	}
	for _, code := range required {
		if _, ok := set[code]; !ok {
			return false, code, nil
		}
	}
	return true, "", nil
}

// --- Permissions CRUD ---

func ListPermissions() ([]models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	var rows []models.Permission
	if err := db.Order("code").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func CreatePermission(code, description string) (*models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if code == "" {
		return nil, errors.New("permission code required")
	}
	p := models.Permission{Code: code, Description: description}
	if err := db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdatePermission(id uint, code *string, description *string) (*models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	var p models.Permission
	if err := db.First(&p, id).Error; err != nil {
		return nil, err
	}
	if code != nil && *code != "" {
		p.Code = *code
	}
	if description != nil {
		p.Description = *description
	}
	if err := db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func DeletePermission(id uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		q, args, err := sqlnamed.Postgres(rbacSQLDeleteRolePermissionsByPermissionID, map[string]interface{}{"permission_id": id})
		if err != nil {
			return err
		}
		if err := tx.Exec(q, args...).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Permission{}, id).Error
	})
}

// --- Roles ---

func ListRoles(withPerms, withParent, withChildren bool) ([]models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	q := db.Order("name")
	if withPerms {
		q = q.Preload("Permissions")
	}
	if withParent {
		q = q.Preload("Parent")
	}
	if withChildren {
		q = q.Preload("Children")
	}
	var rows []models.Role
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func GetRole(id uint, withPerms, withParent, withChildren bool) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	q := db
	if withPerms {
		q = q.Preload("Permissions")
	}
	if withParent {
		q = q.Preload("Parent")
	}
	if withChildren {
		q = q.Preload("Children")
	}
	var r models.Role
	if err := q.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func CreateRole(name, description string, parentID *uint) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("role name required")
	}
	var out models.Role
	err = db.Transaction(func(tx *gorm.DB) error {
		if parentID != nil {
			if err := assertParentExists(tx, *parentID); err != nil {
				return err
			}
		}
		out = models.Role{Name: name, Description: description, ParentID: parentID}
		if err := tx.Create(&out).Error; err != nil {
			return err
		}
		return insertClosureForNewRole(tx, out.ID, parentID)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRole updates fields; if removeParent is true, parent is cleared (parentID ignored).
// If parentID is non-nil and removeParent is false, parent is set (must not create a cycle).
func UpdateRole(id uint, name *string, description *string, parentID *uint, removeParent bool) (*models.Role, error) {
	if removeParent && parentID != nil {
		return nil, errors.New("cannot set both parent_id and remove_parent")
	}
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var r models.Role
		if err := tx.First(&r, id).Error; err != nil {
			return err
		}
		if name != nil && *name != "" {
			r.Name = *name
		}
		if description != nil {
			r.Description = *description
		}
		parentTouched := removeParent || parentID != nil
		if removeParent {
			r.ParentID = nil
		} else if parentID != nil {
			cycle, err := wouldCreateRoleCycle(tx, id, *parentID)
			if err != nil {
				return err
			}
			if cycle {
				return fmt.Errorf("would create cycle: parent %d is under role %d", *parentID, id)
			}
			if err := assertParentExists(tx, *parentID); err != nil {
				return err
			}
			r.ParentID = parentID
		}
		if err := tx.Save(&r).Error; err != nil {
			return err
		}
		if parentTouched {
			return rebuildRoleClosure(tx)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return GetRole(id, true, true, true)
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
		if err := tx.Delete(&models.Role{}, id).Error; err != nil {
			return err
		}
		return rebuildRoleClosure(tx)
	})
}

// SetRolePermissions replaces all permissions on the role using permission codes.
func SetRolePermissions(roleID uint, codes []string) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	var role models.Role
	if err := db.First(&role, roleID).Error; err != nil {
		return nil, err
	}
	var perms []models.Permission
	if len(codes) > 0 {
		if err := db.Where("code IN ?", codes).Find(&perms).Error; err != nil {
			return nil, err
		}
		if len(perms) != len(codes) {
			return nil, fmt.Errorf("unknown permission codes (expected %d, found %d)", len(codes), len(perms))
		}
	}
	if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
		return nil, err
	}
	return GetRole(roleID, true, true, true)
}

// --- User roles ---

func ListUserRoles(userID string) ([]models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	var roleIDs []uint
	if err := db.Model(&models.UserRole{}).Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var roles []models.Role
	if err := db.Preload("Permissions").Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func AssignUserRole(userID string, roleID uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	if userID == "" {
		return errors.New("empty user id")
	}
	var n int64
	if err := db.Model(&models.Role{}).Where("id = ?", roleID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	ur := models.UserRole{UserID: userID, RoleID: roleID}
	return db.FirstOrCreate(&ur, models.UserRole{UserID: userID, RoleID: roleID}).Error
}

func RemoveUserRole(userID string, roleID uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{}).Error
}
