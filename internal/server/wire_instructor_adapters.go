package server

import (
	"context"
	"strings"

	authapp "mycourse-io-be/internal/auth/application"
	authdomain "mycourse-io-be/internal/auth/domain"
	authinfra "mycourse-io-be/internal/auth/infra"
	instapp "mycourse-io-be/internal/instructor/application"
	"mycourse-io-be/internal/instructor/domain"
	mediadomain "mycourse-io-be/internal/media/domain"
	rbacapp "mycourse-io-be/internal/rbac/application"
	rbacdomain "mycourse-io-be/internal/rbac/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/mediaquery"
)

type instructorUserLookup struct{ repo *authinfra.GormUserRepository }

func (l *instructorUserLookup) FindByEmail(ctx context.Context, email string) (*authdomain.User, error) {
	return l.repo.FindByEmail(ctx, email)
}

func (l *instructorUserLookup) FindByID(ctx context.Context, userID string) (*authdomain.User, error) {
	return l.repo.FindByID(ctx, userID)
}

type instructorRoleManager struct {
	rbac   *rbacapp.RBACService
	roleID uint
}

func newInstructorRoleManager(rbac *rbacapp.RBACService) *instructorRoleManager {
	return &instructorRoleManager{rbac: rbac}
}

func (m *instructorRoleManager) InstructorRoleID(ctx context.Context) (uint, error) {
	if m.roleID > 0 {
		return m.roleID, nil
	}
	roles, _, err := m.rbac.ListRoles(ctx, rbacdomain.RoleFilter{Page: 1, PageSize: 50})
	if err != nil {
		return 0, err
	}
	for _, r := range roles {
		if r.Name == domain.RoleNameInstructor {
			m.roleID = r.ID
			return r.ID, nil
		}
	}
	return 0, apperrors.ErrNotFound
}

func (m *instructorRoleManager) AssignInstructorRole(ctx context.Context, userID string) error {
	roleID, err := m.InstructorRoleID(ctx)
	if err != nil {
		return err
	}
	return m.rbac.AssignRoleToUser(ctx, userID, roleID)
}

func (m *instructorRoleManager) RemoveInstructorRole(ctx context.Context, userID string) error {
	roleID, err := m.InstructorRoleID(ctx)
	if err != nil {
		return err
	}
	return m.rbac.RemoveRoleFromUser(ctx, userID, roleID)
}

func (m *instructorRoleManager) UserHasInstructorRole(ctx context.Context, userID string) (bool, error) {
	roles, err := m.rbac.ListRolesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r.Name == domain.RoleNameInstructor {
			return true, nil
		}
	}
	return false, nil
}

type instructorPermissionChecker struct{ rbac *rbacapp.RBACService }

func (p *instructorPermissionChecker) HasPermission(ctx context.Context, userID, action string) bool {
	return p.rbac.HasPermission(ctx, userID, action)
}

type instructorMeCache struct{ auth *authapp.AuthService }

func (c *instructorMeCache) InvalidateUserMeCache(ctx context.Context, userID string) {
	c.auth.InvalidateUserMeCache(ctx, userID)
}

type instructorProfileMediaValidator struct {
	files mediadomain.FileRepository
}

func (v *instructorProfileMediaValidator) ValidateProfilePayload(ctx context.Context, p domain.ProfilePayload) error {
	if id := strings.TrimSpace(p.CVFileID); id != "" {
		if err := v.validatePDF(ctx, id); err != nil {
			return err
		}
	}
	if id := strings.TrimSpace(p.IntroVideoFileID); id != "" {
		if err := v.validateVideo(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (v *instructorProfileMediaValidator) validatePDF(ctx context.Context, fileID string) error {
	f, err := v.files.GetByID(ctx, fileID)
	if err != nil {
		return apperrors.ErrInvalidProfileMediaFile
	}
	if f.Status != constants.FileStatusReady {
		return apperrors.ErrInvalidProfileMediaFile
	}
	mt := strings.ToLower(f.MimeType)
	if strings.Contains(mt, "pdf") || strings.HasSuffix(strings.ToLower(f.Filename), ".pdf") {
		return nil
	}
	return apperrors.ErrInvalidProfileMediaFile
}

func (v *instructorProfileMediaValidator) validateVideo(ctx context.Context, fileID string) error {
	f, err := v.files.GetByID(ctx, fileID)
	if err != nil {
		return apperrors.ErrInvalidProfileMediaFile
	}
	if f.Status != constants.FileStatusReady || f.Kind != constants.FileKindVideo {
		return apperrors.ErrInvalidProfileMediaFile
	}
	return nil
}

type mediaFileURLResolver struct {
	repo mediadomain.FileRepository
}

func (r *mediaFileURLResolver) URLsForFileIDs(ctx context.Context, fileIDs []string) (map[string]string, error) {
	out := make(map[string]string, len(fileIDs))
	for _, id := range fileIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		f, err := r.repo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		out[id] = f.URL
	}
	return out, nil
}

type instructorAvatarHydrator struct{ resolver *mediaFileURLResolver }

func (h *instructorAvatarHydrator) ResolveAvatarURLs(ctx context.Context, fileIDs []string) (map[string]string, error) {
	return mediaquery.HydrateAvatarURLs(ctx, h.resolver, fileIDs)
}

type instructorMediaHydrator struct{ resolver *mediaFileURLResolver }

func (h *instructorMediaHydrator) ResolveMediaFiles(ctx context.Context, fileIDs []string) (map[string]domain.MediaFileReadModel, error) {
	out := make(map[string]domain.MediaFileReadModel, len(fileIDs))
	if h.resolver == nil {
		return out, nil
	}
	urls, err := h.resolver.URLsForFileIDs(ctx, fileIDs)
	if err != nil {
		return nil, err
	}
	for id, url := range urls {
		out[id] = domain.MediaFileReadModel{ID: id, URL: url}
	}
	return out, nil
}

// compile-time interface checks
var (
	_ instapp.UserLookup            = (*instructorUserLookup)(nil)
	_ instapp.InstructorRoleManager = (*instructorRoleManager)(nil)
	_ instapp.PermissionChecker     = (*instructorPermissionChecker)(nil)
	_ instapp.MeCacheInvalidator    = (*instructorMeCache)(nil)
	_ instapp.ProfileMediaValidator = (*instructorProfileMediaValidator)(nil)
	_ instapp.AvatarHydrator        = (*instructorAvatarHydrator)(nil)
	_ instapp.MediaHydrator         = (*instructorMediaHydrator)(nil)
	_ mediaquery.FileURLResolver    = (*mediaFileURLResolver)(nil)
)
