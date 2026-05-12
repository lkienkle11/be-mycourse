// Package application contains the RBAC bounded-context use-case layer.
package application

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/rbac/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

// RBACService provides all RBAC use-cases.
type RBACService struct {
	permRepo     domain.PermissionRepository
	roleRepo     domain.RoleRepository
	userRoleRepo domain.UserRoleRepository
	userPermRepo domain.UserPermissionRepository
}

// NewRBACService constructs an RBACService.
func NewRBACService(
	permRepo domain.PermissionRepository,
	roleRepo domain.RoleRepository,
	userRoleRepo domain.UserRoleRepository,
	userPermRepo domain.UserPermissionRepository,
) *RBACService {
	return &RBACService{
		permRepo: permRepo, roleRepo: roleRepo,
		userRoleRepo: userRoleRepo, userPermRepo: userPermRepo,
	}
}

// --- Permission use-cases ----------------------------------------------------

func (s *RBACService) ListPermissions(ctx context.Context, filter domain.PermissionFilter) ([]domain.Permission, int64, error) {
	return s.permRepo.List(ctx, filter)
}

func (s *RBACService) GetPermission(ctx context.Context, permissionID string) (*domain.Permission, error) {
	return s.permRepo.GetByID(ctx, strings.TrimSpace(permissionID))
}

func (s *RBACService) CreatePermission(ctx context.Context, in domain.CreatePermissionInput) (*domain.Permission, error) {
	id := strings.TrimSpace(in.PermissionID)
	name := strings.TrimSpace(in.PermissionName)
	if id == "" {
		return nil, apperrors.ErrRBACPermissionIDRequired
	}
	if name == "" {
		return nil, apperrors.ErrRBACPermissionNameRequired
	}
	if len(id) > 10 {
		return nil, apperrors.ErrRBACPermissionIDTooLong
	}
	p := &domain.Permission{
		PermissionID: id, PermissionName: name, Description: strings.TrimSpace(in.Description),
	}
	if err := s.permRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return s.permRepo.GetByID(ctx, id)
}

func (s *RBACService) UpdatePermission(ctx context.Context, permissionID string, in domain.UpdatePermissionInput) (*domain.Permission, error) {
	p, err := s.permRepo.GetByID(ctx, strings.TrimSpace(permissionID))
	if err != nil {
		return nil, err
	}
	if in.PermissionName != nil && strings.TrimSpace(*in.PermissionName) != "" {
		p.PermissionName = strings.TrimSpace(*in.PermissionName)
	}
	if in.Description != nil {
		p.Description = strings.TrimSpace(*in.Description)
	}
	if err := s.permRepo.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *RBACService) DeletePermission(ctx context.Context, permissionID string) error {
	return s.permRepo.Delete(ctx, strings.TrimSpace(permissionID))
}

// --- Role use-cases ----------------------------------------------------------

func (s *RBACService) ListRoles(ctx context.Context, filter domain.RoleFilter) ([]domain.Role, int64, error) {
	return s.roleRepo.List(ctx, filter)
}

func (s *RBACService) GetRole(ctx context.Context, id uint, withPermissions bool) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id, withPermissions)
}

func (s *RBACService) CreateRole(ctx context.Context, in domain.CreateRoleInput) (*domain.Role, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperrors.ErrRBACRoleNameRequired
	}
	r := &domain.Role{Name: name, Description: strings.TrimSpace(in.Description)}
	if err := s.roleRepo.Create(ctx, r); err != nil {
		return nil, err
	}
	return s.roleRepo.GetByID(ctx, r.ID, false)
}

func (s *RBACService) UpdateRole(ctx context.Context, id uint, in domain.UpdateRoleInput) (*domain.Role, error) {
	r, err := s.roleRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		r.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		r.Description = strings.TrimSpace(*in.Description)
	}
	if err := s.roleRepo.Save(ctx, r); err != nil {
		return nil, err
	}
	return s.roleRepo.GetByID(ctx, id, false)
}

func (s *RBACService) DeleteRole(ctx context.Context, id uint) error {
	return s.roleRepo.Delete(ctx, id)
}

func (s *RBACService) AssignRolePermissions(ctx context.Context, roleID uint, permissionIDs []string) error {
	return s.roleRepo.AssignPermissions(ctx, roleID, permissionIDs)
}

func (s *RBACService) RemoveRolePermissions(ctx context.Context, roleID uint, permissionIDs []string) error {
	return s.roleRepo.RemovePermissions(ctx, roleID, permissionIDs)
}

func (s *RBACService) SetRolePermissions(ctx context.Context, roleID uint, permissionIDs []string) (*domain.Role, error) {
	if _, err := s.roleRepo.GetByID(ctx, roleID, false); err != nil {
		return nil, err
	}
	if err := s.roleRepo.RemoveAllPermissions(ctx, roleID); err != nil {
		return nil, err
	}
	if len(permissionIDs) > 0 {
		if err := s.roleRepo.AssignPermissions(ctx, roleID, permissionIDs); err != nil {
			return nil, err
		}
	}
	return s.roleRepo.GetByID(ctx, roleID, true)
}

// --- User-role bindings ------------------------------------------------------

func (s *RBACService) ListRolesForUser(ctx context.Context, userID uint) ([]domain.Role, error) {
	return s.userRoleRepo.ListRolesForUser(ctx, userID)
}

func (s *RBACService) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	if userID == 0 {
		return apperrors.ErrRBACInvalidUserID
	}
	return s.userRoleRepo.AssignRole(ctx, userID, roleID)
}

func (s *RBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	return s.userRoleRepo.RemoveRole(ctx, userID, roleID)
}

// --- User-permission bindings ------------------------------------------------

func (s *RBACService) ListPermissionsForUser(ctx context.Context, userID uint) ([]domain.Permission, error) {
	return s.userPermRepo.ListPermissionsForUser(ctx, userID)
}

func (s *RBACService) PermissionCodesForUser(ctx context.Context, userID uint) (map[string]struct{}, error) {
	if userID == 0 {
		return nil, apperrors.ErrRBACInvalidUserID
	}
	return s.userPermRepo.PermissionCodesForUser(ctx, userID)
}

func (s *RBACService) AssignPermissionToUser(ctx context.Context, userID uint, permissionID string) error {
	if userID == 0 {
		return apperrors.ErrRBACInvalidUserID
	}
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return apperrors.ErrRBACPermissionIDRequired
	}
	return s.userPermRepo.AssignPermission(ctx, userID, permissionID)
}

func (s *RBACService) AssignPermissionToUserByName(ctx context.Context, userID uint, permissionName string) error {
	if userID == 0 || strings.TrimSpace(permissionName) == "" {
		return apperrors.ErrRBACUserAndPermissionNameRequired
	}
	return s.userPermRepo.AssignPermissionByName(ctx, userID, permissionName)
}

func (s *RBACService) RemovePermissionFromUser(ctx context.Context, userID uint, permissionID string) error {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return apperrors.ErrRBACPermissionIDRequired
	}
	return s.userPermRepo.RemovePermission(ctx, userID, permissionID)
}

// --- Seed --------------------------------------------------------------------

// SeedDefaultPermissionsAndRoles ensures catalog permissions and baseline roles exist.
func (s *RBACService) SeedDefaultPermissionsAndRoles(ctx context.Context, seedPerms []domain.SeedPermission, db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, e := range seedPerms {
			p := &domain.Permission{
				PermissionID: e.PermissionID, PermissionName: e.PermissionName, Description: e.Description,
			}
			if err := s.permRepo.Upsert(ctx, p); err != nil {
				return err
			}
		}
		roles := []struct{ name, desc string }{
			{"sysadmin", "System-wide administration"},
			{"admin", "Business administration"},
			{"instructor", "Manage and teach courses"},
			{"learner", "Consume learning content"},
		}
		for _, rd := range roles {
			r := &domain.Role{Name: rd.name, Description: rd.desc}
			if err := s.roleRepo.Create(ctx, r); err != nil && !isDuplicateError(err) {
				return err
			}
		}
		return nil
	})
}

// HasPermission returns true when PermissionCodesForUser includes action.
func (s *RBACService) HasPermission(ctx context.Context, userID uint, action string) bool {
	codes, err := s.PermissionCodesForUser(ctx, userID)
	if err != nil {
		return false
	}
	_, ok := codes[action]
	return ok
}

// UserHasAllPermissions implements middleware.PermissionChecker.
// Returns (allGranted, firstMissingPermission, error).
func (s *RBACService) UserHasAllPermissions(userID uint, requiredActions []string) (bool, string, error) {
	codes, err := s.PermissionCodesForUser(context.Background(), userID)
	if err != nil {
		return false, "", err
	}
	for _, action := range requiredActions {
		if _, ok := codes[action]; !ok {
			return false, action, nil
		}
	}
	return true, "", nil
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "UNIQUE constraint")
}
