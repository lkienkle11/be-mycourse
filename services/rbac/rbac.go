package rbac

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/dbschema"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/sqlnamed"
)

var (
	rbacSQLPermissionCodesForUser              string
	rbacSQLDeleteRolePermissionsByPermissionID string
	rbacSQLDeleteRolePermissionsByRoleID       string
	rbacSQLDeleteUserPermissionsByPermissionID string
)

func init() {
	rbacSQLPermissionCodesForUser = fmt.Sprintf(constants.RbacSQLPermissionCodesForUserTmpl,
		dbschema.RBAC.UserRoles(),
		dbschema.RBAC.RolePermissions(),
		dbschema.RBAC.Permissions(),
		dbschema.RBAC.UserPermissions(),
		dbschema.RBAC.Permissions(),
	)
	rbacSQLDeleteRolePermissionsByPermissionID = fmt.Sprintf(constants.RbacSQLDeleteRolePermissionsByPermissionIDTmpl, dbschema.RBAC.RolePermissions())
	rbacSQLDeleteRolePermissionsByRoleID = fmt.Sprintf(constants.RbacSQLDeleteRolePermissionsByRoleIDTmpl, dbschema.RBAC.RolePermissions())
	rbacSQLDeleteUserPermissionsByPermissionID = fmt.Sprintf(constants.RbacSQLDeleteUserPermissionsByPermissionIDTmpl, dbschema.RBAC.UserPermissions())
}

var rbacDB *gorm.DB

func SetRBACDB(db *gorm.DB) {
	rbacDB = db
}

func rbacOrErr() (*gorm.DB, error) {
	if rbacDB == nil {
		return nil, pkgerrors.ErrRBACDatabaseNotConfigured
	}
	return rbacDB, nil
}

// PermissionCodesForUser returns distinct permission_name values from the user's roles (role_permissions)
// plus any direct user_permissions grants. Use with RequirePermission (e.g. constants.AllPermissions.UserRead).
func PermissionCodesForUser(userID uint) (map[string]struct{}, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, pkgerrors.ErrRBACInvalidUserID
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

// UserHasAllPermissions checks permission_name strings (e.g. constants.AllPermissions.UserRead).
func UserHasAllPermissions(userID uint, requiredActions []string) (bool, string, error) {
	if len(requiredActions) == 0 {
		return true, "", nil
	}
	set, err := PermissionCodesForUser(userID)
	if err != nil {
		return false, "", err
	}
	for _, action := range requiredActions {
		if _, ok := set[action]; !ok {
			return false, action, nil
		}
	}
	return true, "", nil
}

// --- User roles ---

func ListUserRoles(userID uint) ([]models.Role, error) {
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

func AssignUserRole(userID uint, roleID uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	if userID == 0 {
		return pkgerrors.ErrRBACInvalidUserID
	}
	var n int64
	if err := db.Model(&models.Role{}).Where("id = ?", roleID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return pkgerrors.ErrNotFound
	}
	ur := models.UserRole{UserID: userID, RoleID: roleID}
	return db.FirstOrCreate(&ur, models.UserRole{UserID: userID, RoleID: roleID}).Error
}

func RemoveUserRole(userID uint, roleID uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{}).Error
}

// --- User direct permissions (supplement role permissions) ---

func ListUserDirectPermissions(userID uint) ([]models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, pkgerrors.ErrRBACInvalidUserID
	}
	var ups []models.UserPermission
	if err := db.Preload("Permission").Where("user_id = ?", userID).Find(&ups).Error; err != nil {
		return nil, err
	}
	out := make([]models.Permission, 0, len(ups))
	for _, up := range ups {
		out = append(out, up.Permission)
	}
	return out, nil
}

func AssignUserPermission(userID uint, permissionID string) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	if userID == 0 {
		return pkgerrors.ErrRBACInvalidUserID
	}
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return pkgerrors.ErrRBACPermissionIDRequired
	}
	var n int64
	if err := db.Model(&models.Permission{}).Where("permission_id = ?", permissionID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return pkgerrors.ErrNotFound
	}
	row := models.UserPermission{UserID: userID, PermissionID: permissionID}
	return db.FirstOrCreate(&row, models.UserPermission{UserID: userID, PermissionID: permissionID}).Error
}

// AssignUserPermissionByPermissionName resolves by permissions.permission_name (colon form).
func AssignUserPermissionByPermissionName(userID uint, permissionName string) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	if userID == 0 || strings.TrimSpace(permissionName) == "" {
		return pkgerrors.ErrRBACUserAndPermissionNameRequired
	}
	var p models.Permission
	if err := db.Where("permission_name = ?", strings.TrimSpace(permissionName)).First(&p).Error; err != nil {
		return pkgerrors.MapRecordNotFound(err)
	}
	return AssignUserPermission(userID, p.PermissionID)
}

func RemoveUserPermission(userID uint, permissionID string) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return pkgerrors.ErrRBACPermissionIDRequired
	}
	return db.Where("user_id = ? AND permission_id = ?", userID, permissionID).Delete(&models.UserPermission{}).Error
}
