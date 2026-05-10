package rbac

import (
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	errfuncdb "mycourse-io-be/pkg/errors_func/db"
	"mycourse-io-be/pkg/sqlnamed"
)

// --- Permissions CRUD ---

var allowedPermissionSortCols = map[string]string{
	"permission_id":   "permission_id",
	"permission_name": "permission_name",
	"description":     "description",
	"created_at":      "created_at",
}

var allowedPermissionSearchCols = map[string]string{
	"permission_id":   "permission_id",
	"permission_name": "permission_name",
	"description":     "description",
}

func listPermissionsOrderExpr(p dto.ListPermissionsParams) string {
	col := "permission_id"
	if c, ok := allowedPermissionSortCols[p.SortBy]; ok {
		col = c
	}
	if p.SortOrder == "desc" {
		return col + " DESC"
	}
	return col + " ASC"
}

// ListPermissions returns a filtered, sorted, paginated page of permissions and the total
// count of matching rows (before pagination).
func ListPermissions(p dto.ListPermissionsParams) ([]models.Permission, int64, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, 0, err
	}
	q := db.Model(&models.Permission{})
	if p.SearchBy != "" && p.SearchData != "" {
		if c, ok := allowedPermissionSearchCols[p.SearchBy]; ok {
			q = q.Where(c+" ILIKE ?", "%"+p.SearchData+"%")
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.Permission
	if err := q.Order(listPermissionsOrderExpr(p)).Offset(p.Offset).Limit(p.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func CreatePermission(permissionID, permissionName, description string) (*models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	permissionID = strings.TrimSpace(permissionID)
	permissionName = strings.TrimSpace(permissionName)
	if permissionID == "" {
		return nil, pkgerrors.ErrRBACPermissionIDRequired
	}
	if permissionName == "" {
		return nil, pkgerrors.ErrRBACPermissionNameRequired
	}
	if len(permissionID) > 10 {
		return nil, pkgerrors.ErrRBACPermissionIDTooLong
	}
	if len(permissionName) > 50 {
		return nil, pkgerrors.ErrRBACPermissionNameTooLong
	}
	p := models.Permission{
		PermissionID:   permissionID,
		PermissionName: permissionName,
		Description:    description,
	}
	if err := db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func rbacApplyRenamedPermissionID(db *gorm.DB, p *models.Permission, newPermissionID *string) error {
	if newPermissionID == nil {
		return nil
	}
	n := strings.TrimSpace(*newPermissionID)
	if n == "" || n == p.PermissionID {
		return nil
	}
	if len(n) > 10 {
		return pkgerrors.ErrRBACPermissionIDTooLong
	}
	if err := db.Model(&models.Permission{}).Where("permission_id = ?", p.PermissionID).Update("permission_id", n).Error; err != nil {
		return err
	}
	p.PermissionID = n
	return nil
}

func rbacApplyPermissionNameUpdate(p *models.Permission, permissionName *string) error {
	if permissionName == nil {
		return nil
	}
	v := strings.TrimSpace(*permissionName)
	if v == "" {
		return nil
	}
	if len(v) > 50 {
		return pkgerrors.ErrRBACPermissionNameTooLong
	}
	p.PermissionName = v
	return nil
}

// UpdatePermission updates a permission row. If newPermissionID is set and differs from permissionID,
// permissions.permission_id is changed (FKs use ON UPDATE CASCADE).
func UpdatePermission(permissionID string, newPermissionID *string, permissionName *string, description *string) (*models.Permission, error) {
	db, err := rbacOrErr()
	if err != nil {
		return nil, err
	}
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return nil, pkgerrors.ErrRBACPermissionIDRequired
	}
	var p models.Permission
	if err := db.Where("permission_id = ?", permissionID).First(&p).Error; err != nil {
		return nil, errfuncdb.MapRecordNotFound(err)
	}
	if err := rbacApplyRenamedPermissionID(db, &p, newPermissionID); err != nil {
		return nil, err
	}
	if err := rbacApplyPermissionNameUpdate(&p, permissionName); err != nil {
		return nil, err
	}
	if description != nil {
		p.Description = *description
	}
	if err := db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func DeletePermission(permissionID string) error {
	db, err := rbacOrErr()
	if err != nil {
		return err
	}
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return pkgerrors.ErrRBACPermissionIDRequired
	}
	return db.Transaction(func(tx *gorm.DB) error {
		q, args, err := sqlnamed.Postgres(rbacSQLDeleteRolePermissionsByPermissionID, map[string]interface{}{"permission_id": permissionID})
		if err != nil {
			return err
		}
		if err := tx.Exec(q, args...).Error; err != nil {
			return err
		}
		q2, args2, err := sqlnamed.Postgres(rbacSQLDeleteUserPermissionsByPermissionID, map[string]interface{}{"permission_id": permissionID})
		if err != nil {
			return err
		}
		if err := tx.Exec(q2, args2...).Error; err != nil {
			return err
		}
		return tx.Where("permission_id = ?", permissionID).Delete(&models.Permission{}).Error
	})
}
