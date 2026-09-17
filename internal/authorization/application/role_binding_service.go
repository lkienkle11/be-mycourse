package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/authorization/domain"
	"mycourse-io-be/internal/shared/timex"
)

// RoleBindingService is the write path for resource-scoped role bindings
// (authorization/role-binding-management): assigning and revoking a named role for a batch of
// principals on one resource or on every resource of a type (domain.WildcardResourceID). It
// never gates on a role name itself — see requireRoleBindingManager — only on whether the
// issuer may manage grants for the target resource, exactly like GrantService.
type RoleBindingService struct {
	registry *Registry
	repo     domain.RoleBindingRepository
	now      func() int64
}

func NewRoleBindingService(registry *Registry, repo domain.RoleBindingRepository) *RoleBindingService {
	return &RoleBindingService{registry: registry, repo: repo, now: timex.NowUnix}
}

// Assign grants roleName to every principal in command.PrincipalUserIDs, in one batched write.
func (s *RoleBindingService) Assign(ctx context.Context, command domain.RoleBindingAssignment) error {
	command.Issuer.ID = strings.TrimSpace(command.Issuer.ID)
	command.PrincipalUserIDs = uniqueNonEmpty(command.PrincipalUserIDs)
	command.RoleName = strings.TrimSpace(command.RoleName)
	command.Resource.Type = strings.TrimSpace(command.Resource.Type)
	command.Resource.ID = strings.TrimSpace(command.Resource.ID)
	if err := validateRoleBindingShape(command.Issuer, command.PrincipalUserIDs, command.Resource); err != nil {
		return err
	}
	if command.RoleName == "" {
		return domain.ErrInvalidRoleBinding
	}
	if err := s.requireRoleBindingManager(command.Issuer, command.PrincipalUserIDs, command.Resource, command.Context); err != nil {
		return err
	}
	return s.repo.AssignRoleBindings(ctx, command.PrincipalUserIDs, command.RoleName, command.Resource, command.Issuer.ID, s.now())
}

// Revoke ends command.RoleName for every principal in command.PrincipalUserIDs on
// command.Resource, in one batched write. Revoking an already-revoked binding is a no-op:
// RevokeRoleBindings' own "revoked_at IS NULL" filter simply matches nothing further for that
// principal. command.RoleName may be left empty to revoke every active role binding for the
// principal(s) on the resource regardless of role, for callers removing a principal's
// membership entirely rather than one specific role.
func (s *RoleBindingService) Revoke(ctx context.Context, command domain.RoleBindingRevocation) error {
	command.Issuer.ID = strings.TrimSpace(command.Issuer.ID)
	command.PrincipalUserIDs = uniqueNonEmpty(command.PrincipalUserIDs)
	command.RoleName = strings.TrimSpace(command.RoleName)
	command.Resource.Type = strings.TrimSpace(command.Resource.Type)
	command.Resource.ID = strings.TrimSpace(command.Resource.ID)
	if err := validateRoleBindingShape(command.Issuer, command.PrincipalUserIDs, command.Resource); err != nil {
		return err
	}
	if err := s.requireRoleBindingManager(command.Issuer, command.PrincipalUserIDs, command.Resource, command.Context); err != nil {
		return err
	}
	return s.repo.RevokeRoleBindings(ctx, domain.RoleBindingQuery{
		PrincipalUserIDs: command.PrincipalUserIDs, RoleName: command.RoleName, Resource: command.Resource,
	}, s.now())
}

// RevokeResource revokes every active role binding on resource, regardless of principal or
// role name, with no CanManageGrants check — reserved for domain lifecycle cleanup after a
// resource itself is deleted, mirroring GrantService.RevokeResource's own exemption from the
// boundary-permission/CanManageGrants check (there is no acting issuer to check in that case).
func (s *RoleBindingService) RevokeResource(ctx context.Context, resource domain.Resource) error {
	resource.Type = strings.TrimSpace(resource.Type)
	resource.ID = strings.TrimSpace(resource.ID)
	if resource.Type == "" || resource.ID == "" {
		return domain.ErrInvalidRoleBinding
	}
	return s.repo.RevokeRoleBindings(ctx, domain.RoleBindingQuery{Resource: resource}, s.now())
}

// List returns every active role binding matching query, for display purposes (e.g. listing a
// resource's collaborators, or building an exclusion set for a candidate picker). Unlike
// Assign/Revoke, it has no CanManageGrants gate: reading a list to display is not a mutation,
// and the caller is expected to have already gated the read itself (e.g. Course's own
// requireCourseAction check before calling this).
func (s *RoleBindingService) List(ctx context.Context, query domain.RoleBindingQuery) ([]domain.RoleBinding, error) {
	query.RoleName = strings.TrimSpace(query.RoleName)
	query.Resource.Type = strings.TrimSpace(query.Resource.Type)
	query.Resource.ID = strings.TrimSpace(query.Resource.ID)
	return s.repo.ListRoleBindings(ctx, query)
}

// validateRoleBindingShape checks the fields common to Assign and Revoke. It does not check
// RoleName: Assign requires a non-empty one itself, while Revoke allows an empty one (meaning
// "any role") by design.
func validateRoleBindingShape(issuer domain.Principal, principalUserIDs []string, resource domain.Resource) error {
	if issuer.Type != domain.PrincipalTypeUser || issuer.ID == "" {
		return domain.ErrInvalidRoleBinding
	}
	if len(principalUserIDs) == 0 {
		return domain.ErrInvalidRoleBinding
	}
	if resource.Type == "" || resource.ID == "" {
		return domain.ErrInvalidRoleBinding
	}
	return nil
}

// requireRoleBindingManager mirrors GrantService.requireManager/requireBoundaryPermissions'
// shape (a provider-level CanManageGrants check plus a global boundary-permission check over
// every grantable action of the resource type) as its own copy, rather than a shared call, so
// GrantService/Authorizer/Registry stay unchanged by this change.
func (s *RoleBindingService) requireRoleBindingManager(
	issuer domain.Principal,
	targetPrincipalIDs []string,
	resource domain.Resource,
	policyContext domain.EvaluationContext,
) error {
	provider, ok := s.registry.Provider(resource.Type)
	if !ok {
		return domain.ErrInvalidRoleBinding
	}
	for _, actionName := range s.registry.GrantableActions(resource.Type) {
		action, ok := s.registry.Action(actionName)
		if !ok {
			return domain.ErrInvalidRoleBinding
		}
		if _, held := issuer.GlobalPermissions[action.BoundaryPermission]; !held {
			return domain.ErrGrantManagementForbidden
		}
	}
	allowed, err := provider.CanManageGrants(domain.GrantManagementRequest{
		Issuer: issuer, TargetPrincipalIDs: targetPrincipalIDs, Resource: resource, Context: policyContext,
	})
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrGrantManagementForbidden
	}
	return nil
}
