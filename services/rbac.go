package services

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/dbschema"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/sqlnamed"
)

// Native RBAC SQL (named value params :name via sqlnamed.Postgres; table names from dbschema in init).
const (
	rbacSQLPermissionCodesForUserTmpl = `
SELECT DISTINCT cc FROM (
  SELECT p.action AS cc
  FROM %s ur
  INNER JOIN %s rp ON rp.role_id = ur.role_id
  INNER JOIN %s p ON p.id = rp.permission_id
  WHERE ur.user_id = :user_id
  UNION
  SELECT p.action
  FROM %s up
  INNER JOIN %s p ON p.id = up.permission_id
  WHERE up.user_id = :user_id
) AS _
`
	rbacSQLDeleteRolePermissionsByPermissionIDTmpl = `
DELETE FROM %s WHERE permission_id = :permission_id
`
	rbacSQLDeleteRolePermissionsByRoleIDTmpl = `
DELETE FROM %s WHERE role_id = :role_id
`
	rbacSQLDeleteUserPermissionsByPermissionIDTmpl = `
DELETE FROM %s WHERE permission_id = :permission_id
`
)

var (
	rbacSQLPermissionCodesForUser              string
	rbacSQLDeleteRolePermissionsByPermissionID string
	rbacSQLDeleteRolePermissionsByRoleID       string
	rbacSQLDeleteUserPermissionsByPermissionID string
)

func init() {
	rbacSQLPermissionCodesForUser = fmt.Sprintf(rbacSQLPermissionCodesForUserTmpl,
		dbschema.RBAC.UserRoles(),
		dbschema.RBAC.RolePermissions(),
		dbschema.RBAC.Permissions(),
		dbschema.RBAC.UserPermissions(),
		dbschema.RBAC.Permissions(),
	)
	rbacSQLDeleteRolePermissionsByPermissionID = fmt.Sprintf(rbacSQLDeleteRolePermissionsByPermissionIDTmpl, dbschema.RBAC.RolePermissions())
	rbacSQLDeleteRolePermissionsByRoleID = fmt.Sprintf(rbacSQLDeleteRolePermissionsByRoleIDTmpl, dbschema.RBAC.RolePermissions())
	rbacSQLDeleteUserPermissionsByPermissionID = fmt.Sprintf(rbacSQLDeleteUserPermissionsByPermissionIDTmpl, dbschema.RBAC.UserPermissions())
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

// PermissionCodesForUser returns distinct permission action values from the user's roles (role_permissions)
// plus any direct user_permissions grants. Use catalog field strings with RequirePermission (e.g. constants.CodeUser.Read).
func PermissionCodesForUser(userID uint) (map[string]struct{}, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, errors.New("invalid user id")
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

// UserHasAllPermissions checks action strings (e.g. constants.CodeUser.Read).
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

// --- Permissions CRUD ---

// allowedPermissionSortCols maps client-supplied sort_by values to safe SQL column names.
// Only columns in this map are accepted — anything else falls back to the default ("code").
var allowedPermissionSortCols = map[string]string{
	"id":          "id",
	"code":        "code",
	"description": "description",
	"created_at":  "created_at",
}

// allowedPermissionSearchCols maps client-supplied search_by values to safe SQL column names.
var allowedPermissionSearchCols = map[string]string{
	"code":        "code",
	"description": "description",
}

// ListPermissionsParams carries the validated filter values from the handler.
type ListPermissionsParams struct {
	Offset     int
	Limit      int
	SortBy     string
	SortOrder  string // "asc" | "desc"
	SearchBy   string
	SearchData string
}

// ListPermissions returns a filtered, sorted, paginated page of permissions and the total
// count of matching rows (before pagination).
func ListPermissions(p ListPermissionsParams) ([]models.Permission, int64, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, 0, err
	}

	q := db.Model(&models.Permission{})

	// Apply search predicate when both search_by and search_data are present.
	if p.SearchBy != "" && p.SearchData != "" {
		if col, ok := allowedPermissionSearchCols[p.SearchBy]; ok {
			q = q.Where(col+" ILIKE ?", "%"+p.SearchData+"%")
		}
	}

	// Count total matching rows (without LIMIT/OFFSET).
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build ORDER BY — default to "code ASC" when sort_by is absent or not whitelisted.
	orderCol := "code"
	if col, ok := allowedPermissionSortCols[p.SortBy]; ok {
		orderCol = col
	}
	orderDir := "ASC"
	if p.SortOrder == "desc" {
		orderDir = "DESC"
	}

	var rows []models.Permission
	if err := q.Order(orderCol + " " + orderDir).
		Offset(p.Offset).
		Limit(p.Limit).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func CreatePermission(code, description string, action *string) (*models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if code == "" {
		return nil, errors.New("permission code required")
	}
	permissionAction := code
	if action != nil && strings.TrimSpace(*action) != "" {
		permissionAction = strings.TrimSpace(*action)
	}
	p := models.Permission{Code: code, Action: permissionAction, Description: description}
	if err := db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdatePermission(id uint, code *string, action *string, description *string) (*models.Permission, error) {
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
	if action != nil {
		if strings.TrimSpace(*action) != "" {
			p.Action = strings.TrimSpace(*action)
		}
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
		q2, args2, err := sqlnamed.Postgres(rbacSQLDeleteUserPermissionsByPermissionID, map[string]interface{}{"permission_id": id})
		if err != nil {
			return err
		}
		if err := tx.Exec(q2, args2...).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Permission{}, id).Error
	})
}

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
		return nil, err
	}
	return &r, nil
}

func CreateRole(name, description string) (*models.Role, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("role name required")
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
		return nil, err
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
	return GetRole(roleID, true)
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
		return errors.New("invalid user id")
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
		return nil, errors.New("invalid user id")
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

func AssignUserPermission(userID uint, permissionID uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	if userID == 0 {
		return errors.New("invalid user id")
	}
	var n int64
	if err := db.Model(&models.Permission{}).Where("id = ?", permissionID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	row := models.UserPermission{UserID: userID, PermissionID: permissionID}
	return db.FirstOrCreate(&row, models.UserPermission{UserID: userID, PermissionID: permissionID}).Error
}

func AssignUserPermissionByCode(userID uint, code string) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	if userID == 0 || code == "" {
		return errors.New("user id and permission code required")
	}
	var p models.Permission
	if err := db.Where("code = ?", code).First(&p).Error; err != nil {
		return err
	}
	return AssignUserPermission(userID, p.ID)
}

func RemoveUserPermission(userID uint, permissionID uint) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	return db.Where("user_id = ? AND permission_id = ?", userID, permissionID).Delete(&models.UserPermission{}).Error
}
