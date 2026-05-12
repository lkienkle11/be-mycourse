// Package infra contains the RBAC bounded-context infrastructure (GORM repositories + raw SQL).
package infra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/rbac/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/utils"
)

// Pre-built SQL strings (filled once from table names).
var (
	sqlPermissionCodesForUser              string
	sqlDeleteRolePermsByPermissionID       string
	sqlDeleteRolePermsByRoleID             string
	sqlDeleteUserPermsByPermissionID       string
)

func init() {
	sqlPermissionCodesForUser = fmt.Sprintf(RbacSQLPermissionCodesForUserTmpl,
		constants.TableRBACUserRoles,
		constants.TableRBACRolePermissions,
		constants.TableRBACPermissions,
		constants.TableRBACUserPermissions,
		constants.TableRBACPermissions,
	)
	sqlDeleteRolePermsByPermissionID = fmt.Sprintf(RbacSQLDeleteRolePermissionsByPermissionIDTmpl, constants.TableRBACRolePermissions)
	sqlDeleteRolePermsByRoleID = fmt.Sprintf(RbacSQLDeleteRolePermissionsByRoleIDTmpl, constants.TableRBACRolePermissions)
	sqlDeleteUserPermsByPermissionID = fmt.Sprintf(RbacSQLDeleteUserPermissionsByPermissionIDTmpl, constants.TableRBACUserPermissions)
}

// --- GORM row types ----------------------------------------------------------

type permissionRow struct {
	PermissionID   string    `gorm:"column:permission_id;primaryKey;size:10"`
	PermissionName string    `gorm:"column:permission_name;uniqueIndex;size:50;not null"`
	Description    string    `gorm:"size:512"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (permissionRow) TableName() string { return constants.TableRBACPermissions }

type roleRow struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"uniqueIndex;size:64;not null"`
	Description string    `gorm:"size:512"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (roleRow) TableName() string { return constants.TableRBACRoles }

type rolePermissionRow struct {
	RoleID       uint   `gorm:"column:role_id;primaryKey"`
	PermissionID string `gorm:"column:permission_id;primaryKey;size:10"`
}

func (rolePermissionRow) TableName() string { return constants.TableRBACRolePermissions }

type userRoleRow struct {
	UserID uint `gorm:"primaryKey"`
	RoleID uint `gorm:"primaryKey"`
}

func (userRoleRow) TableName() string { return constants.TableRBACUserRoles }

type userPermissionRow struct {
	UserID       uint   `gorm:"primaryKey"`
	PermissionID string `gorm:"primaryKey;size:10"`
}

func (userPermissionRow) TableName() string { return constants.TableRBACUserPermissions }

// --- mappers -----------------------------------------------------------------

func rowToPermission(r *permissionRow) domain.Permission {
	return domain.Permission{
		PermissionID: r.PermissionID, PermissionName: r.PermissionName,
		Description: r.Description, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func permissionToRow(p *domain.Permission) *permissionRow {
	return &permissionRow{
		PermissionID: p.PermissionID, PermissionName: p.PermissionName,
		Description: p.Description, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func rowToRole(r *roleRow, perms []permissionRow) domain.Role {
	role := domain.Role{ID: r.ID, Name: r.Name, Description: r.Description, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	if len(perms) > 0 {
		role.Permissions = make([]domain.Permission, len(perms))
		for i := range perms {
			role.Permissions[i] = rowToPermission(&perms[i])
		}
	}
	return role
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrNotFound
	}
	return err
}

// --- GormPermissionRepository ------------------------------------------------

// GormPermissionRepository implements domain.PermissionRepository.
type GormPermissionRepository struct{ db *gorm.DB }

func NewGormPermissionRepository(db *gorm.DB) *GormPermissionRepository {
	return &GormPermissionRepository{db: db}
}

func (r *GormPermissionRepository) List(ctx context.Context, filter domain.PermissionFilter) ([]domain.Permission, int64, error) {
	q := r.db.WithContext(ctx).Model(&permissionRow{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pageParams(filter.Page, filter.PageSize)
	var rows []permissionRow
	if err := q.Order("permission_id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Permission, len(rows))
	for i := range rows {
		out[i] = rowToPermission(&rows[i])
	}
	return out, total, nil
}

func (r *GormPermissionRepository) GetByID(ctx context.Context, permissionID string) (*domain.Permission, error) {
	var row permissionRow
	if err := r.db.WithContext(ctx).Where("permission_id = ?", permissionID).First(&row).Error; err != nil {
		return nil, mapNotFound(err)
	}
	p := rowToPermission(&row)
	return &p, nil
}

func (r *GormPermissionRepository) Create(ctx context.Context, p *domain.Permission) error {
	row := permissionToRow(p)
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormPermissionRepository) Save(ctx context.Context, p *domain.Permission) error {
	row := permissionToRow(p)
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormPermissionRepository) Delete(ctx context.Context, permissionID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(sqlDeleteRolePermsByPermissionID, map[string]any{"permission_id": permissionID}).Error; err != nil {
			return err
		}
		if err := tx.Exec(sqlDeleteUserPermsByPermissionID, map[string]any{"permission_id": permissionID}).Error; err != nil {
			return err
		}
		return tx.Where("permission_id = ?", permissionID).Delete(&permissionRow{}).Error
	})
}

func (r *GormPermissionRepository) Upsert(ctx context.Context, p *domain.Permission) error {
	row := permissionToRow(p)
	return r.db.WithContext(ctx).
		Where("permission_id = ?", row.PermissionID).
		Assign(row).
		FirstOrCreate(row).Error
}

// --- GormRoleRepository ------------------------------------------------------

// GormRoleRepository implements domain.RoleRepository.
type GormRoleRepository struct{ db *gorm.DB }

func NewGormRoleRepository(db *gorm.DB) *GormRoleRepository {
	return &GormRoleRepository{db: db}
}

func (r *GormRoleRepository) List(ctx context.Context, filter domain.RoleFilter) ([]domain.Role, int64, error) {
	q := r.db.WithContext(ctx).Model(&roleRow{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pageParams(filter.Page, filter.PageSize)
	var rows []roleRow
	if err := q.Order("name ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Role, len(rows))
	for i := range rows {
		var perms []permissionRow
		if filter.WithPermissions {
			perms, _ = r.loadRolePermissions(ctx, rows[i].ID)
		}
		out[i] = rowToRole(&rows[i], perms)
	}
	return out, total, nil
}

func (r *GormRoleRepository) GetByID(ctx context.Context, id uint, withPermissions bool) (*domain.Role, error) {
	var row roleRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	var perms []permissionRow
	if withPermissions {
		perms, _ = r.loadRolePermissions(ctx, id)
	}
	role := rowToRole(&row, perms)
	return &role, nil
}

func (r *GormRoleRepository) Create(ctx context.Context, role *domain.Role) error {
	row := &roleRow{Name: role.Name, Description: role.Description}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	role.ID = row.ID
	role.CreatedAt = row.CreatedAt
	role.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRoleRepository) Save(ctx context.Context, role *domain.Role) error {
	row := &roleRow{ID: role.ID, Name: role.Name, Description: role.Description}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRoleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(sqlDeleteRolePermsByRoleID, map[string]any{"role_id": id}).Error; err != nil {
			return err
		}
		return tx.Delete(&roleRow{}, id).Error
	})
}

func (r *GormRoleRepository) AssignPermissions(ctx context.Context, roleID uint, permissionIDs []string) error {
	rows := make([]rolePermissionRow, 0, len(permissionIDs))
	for _, pid := range permissionIDs {
		rows = append(rows, rolePermissionRow{RoleID: roleID, PermissionID: strings.TrimSpace(pid)})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *GormRoleRepository) RemovePermissions(ctx context.Context, roleID uint, permissionIDs []string) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id IN ?", roleID, permissionIDs).
		Delete(&rolePermissionRow{}).Error
}

func (r *GormRoleRepository) RemoveAllPermissions(ctx context.Context, roleID uint) error {
	return r.db.WithContext(ctx).Exec(sqlDeleteRolePermsByRoleID, map[string]any{"role_id": roleID}).Error
}

func (r *GormRoleRepository) loadRolePermissions(ctx context.Context, roleID uint) ([]permissionRow, error) {
	var rows []permissionRow
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN "+constants.TableRBACRolePermissions+" rp ON rp.permission_id = permissions.permission_id").
		Where("rp.role_id = ?", roleID).
		Find(&rows).Error
	return rows, err
}

// --- GormUserRoleRepository --------------------------------------------------

// GormUserRoleRepository implements domain.UserRoleRepository.
type GormUserRoleRepository struct{ db *gorm.DB }

func NewGormUserRoleRepository(db *gorm.DB) *GormUserRoleRepository {
	return &GormUserRoleRepository{db: db}
}

func (r *GormUserRoleRepository) ListRolesForUser(ctx context.Context, userID uint) ([]domain.Role, error) {
	var rows []roleRow
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN "+constants.TableRBACUserRoles+" ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", userID).
		Order("roles.name ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Role, len(rows))
	for i := range rows {
		out[i] = rowToRole(&rows[i], nil)
	}
	return out, nil
}

func (r *GormUserRoleRepository) AssignRole(ctx context.Context, userID, roleID uint) error {
	row := userRoleRow{UserID: userID, RoleID: roleID}
	return r.db.WithContext(ctx).FirstOrCreate(&row, userRoleRow{UserID: userID, RoleID: roleID}).Error
}

func (r *GormUserRoleRepository) RemoveRole(ctx context.Context, userID, roleID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&userRoleRow{}).Error
}

// --- GormUserPermissionRepository --------------------------------------------

// GormUserPermissionRepository implements domain.UserPermissionRepository.
type GormUserPermissionRepository struct{ db *gorm.DB }

func NewGormUserPermissionRepository(db *gorm.DB) *GormUserPermissionRepository {
	return &GormUserPermissionRepository{db: db}
}

func (r *GormUserPermissionRepository) ListPermissionsForUser(ctx context.Context, userID uint) ([]domain.Permission, error) {
	var rows []permissionRow
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN "+constants.TableRBACUserPermissions+" up ON up.permission_id = permissions.permission_id").
		Where("up.user_id = ?", userID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Permission, len(rows))
	for i := range rows {
		out[i] = rowToPermission(&rows[i])
	}
	return out, nil
}

func (r *GormUserPermissionRepository) PermissionCodesForUser(ctx context.Context, userID uint) (map[string]struct{}, error) {
	q, args, err := utils.Postgres(sqlPermissionCodesForUser, map[string]any{"user_id": userID})
	if err != nil {
		return nil, err
	}
	var codes []string
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&codes).Error; err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		out[c] = struct{}{}
	}
	return out, nil
}

func (r *GormUserPermissionRepository) AssignPermission(ctx context.Context, userID uint, permissionID string) error {
	row := userPermissionRow{UserID: userID, PermissionID: strings.TrimSpace(permissionID)}
	return r.db.WithContext(ctx).FirstOrCreate(&row, row).Error
}

func (r *GormUserPermissionRepository) AssignPermissionByName(ctx context.Context, userID uint, permissionName string) error {
	var p permissionRow
	if err := r.db.WithContext(ctx).Where("permission_name = ?", strings.TrimSpace(permissionName)).First(&p).Error; err != nil {
		return mapNotFound(err)
	}
	return r.AssignPermission(ctx, userID, p.PermissionID)
}

func (r *GormUserPermissionRepository) RemovePermission(ctx context.Context, userID uint, permissionID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND permission_id = ?", userID, strings.TrimSpace(permissionID)).
		Delete(&userPermissionRow{}).Error
}

// --- helpers -----------------------------------------------------------------

func pageParams(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return page, pageSize
}
