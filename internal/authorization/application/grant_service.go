package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"

	"mycourse-io-be/internal/authorization/domain"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/timex"
)

type GrantCommand struct {
	Issuer            domain.Principal
	TargetPrincipalID string
	Action            string
	Resource          domain.Resource
	Effect            domain.Effect
	Conditions        []domain.Condition
	ValidFrom         *int64
	ExpiresAt         *int64
	Context           domain.EvaluationContext
}

type ReplaceGrantsCommand struct {
	Issuer             domain.Principal
	TargetPrincipalIDs []string
	Actions            []string
	Resource           domain.Resource
	Effect             domain.Effect
	Conditions         []domain.Condition
	ValidFrom          *int64
	ExpiresAt          *int64
	Context            domain.EvaluationContext
}

type GrantService struct {
	registry *Registry
	repo     domain.GrantRepository
	now      func() int64
}

func NewGrantService(registry *Registry, repo domain.GrantRepository) *GrantService {
	return &GrantService{registry: registry, repo: repo, now: timex.NowUnix}
}

func (s *GrantService) SyncActions(ctx context.Context) error {
	return s.repo.UpsertActions(ctx, s.registry.Actions())
}

// Grant appends one independent policy statement; it does not replace or deduplicate matching statements.
func (s *GrantService) Grant(ctx context.Context, command GrantCommand) (*domain.Grant, error) {
	replacement := normalizeReplaceCommand(ReplaceGrantsCommand{
		Issuer: command.Issuer, TargetPrincipalIDs: []string{command.TargetPrincipalID},
		Actions: []string{command.Action}, Resource: command.Resource, Effect: command.Effect,
		Conditions: command.Conditions, ValidFrom: command.ValidFrom, ExpiresAt: command.ExpiresAt,
		Context: command.Context,
	})
	if len(replacement.TargetPrincipalIDs) != 1 || len(replacement.Actions) != 1 {
		return nil, domain.ErrInvalidGrant
	}
	if err := s.validateManagement(replacement); err != nil {
		return nil, err
	}
	if err := s.requireManager(replacement.Issuer, replacement.TargetPrincipalIDs, replacement.Resource, replacement.Context); err != nil {
		return nil, err
	}
	now := s.now()
	grant := &domain.Grant{
		PrincipalUserID: replacement.TargetPrincipalIDs[0], ActionName: replacement.Actions[0],
		ResourceType: replacement.Resource.Type, ResourceID: replacement.Resource.ID,
		Effect: command.Effect, Conditions: command.Conditions, ValidFrom: command.ValidFrom,
		ExpiresAt: command.ExpiresAt, GrantedByUserID: replacement.Issuer.ID, CreatedAt: now,
	}
	if err := s.repo.Create(ctx, grant); err != nil {
		return nil, err
	}
	s.logMutation(ctx, "grant", replacement.Issuer.ID, replacement.Resource, replacement.TargetPrincipalIDs, replacement.Actions)
	return grant, nil
}

func (s *GrantService) Replace(ctx context.Context, command ReplaceGrantsCommand) error {
	command = normalizeReplaceCommand(command)
	if len(command.TargetPrincipalIDs) != 1 {
		return domain.ErrInvalidGrant
	}
	return s.ReplaceMany(ctx, command)
}

func (s *GrantService) ReplaceMany(ctx context.Context, command ReplaceGrantsCommand) error {
	command = normalizeReplaceCommand(command)
	if err := s.validateManagement(command); err != nil {
		return err
	}
	if err := s.requireManager(command.Issuer, command.TargetPrincipalIDs, command.Resource, command.Context); err != nil {
		return err
	}
	managedActions := s.registry.GrantableActions(command.Resource.Type)
	if len(managedActions) == 0 {
		return domain.ErrInvalidGrant
	}
	// The repository only revokes grants matching command.Effect (ALLOW for every
	// current caller), so an existing DENY grant on the same principal, resource,
	// and action is left untouched by this replace call.
	if err := s.repo.ReplaceMany(ctx, domain.GrantReplacement{
		PrincipalUserIDs: command.TargetPrincipalIDs,
		ManagedActions:   managedActions,
		Actions:          command.Actions,
		Resource:         command.Resource,
		Effect:           command.Effect,
		Conditions:       command.Conditions,
		ValidFrom:        command.ValidFrom,
		ExpiresAt:        command.ExpiresAt,
		GrantedByUserID:  command.Issuer.ID,
	}, s.now()); err != nil {
		return err
	}
	s.logMutation(ctx, "replace", command.Issuer.ID, command.Resource, command.TargetPrincipalIDs, command.Actions)
	return nil
}

func (s *GrantService) Revoke(ctx context.Context, command GrantCommand) error {
	replacement := normalizeReplaceCommand(ReplaceGrantsCommand{
		Issuer: command.Issuer, TargetPrincipalIDs: []string{command.TargetPrincipalID},
		Actions: []string{command.Action}, Resource: command.Resource, Effect: command.Effect,
		Context: command.Context,
	})
	if len(replacement.TargetPrincipalIDs) != 1 || len(replacement.Actions) != 1 {
		return domain.ErrInvalidGrant
	}
	if err := s.validateManagement(replacement); err != nil {
		return err
	}
	if err := s.requireManager(replacement.Issuer, replacement.TargetPrincipalIDs, replacement.Resource, replacement.Context); err != nil {
		return err
	}
	if err := s.repo.Revoke(ctx, domain.GrantQuery{
		PrincipalUserIDs: replacement.TargetPrincipalIDs, ActionNames: replacement.Actions,
		Resource: replacement.Resource, Effects: []domain.Effect{replacement.Effect},
	}, s.now()); err != nil {
		return err
	}
	s.logMutation(ctx, "revoke", replacement.Issuer.ID, replacement.Resource, replacement.TargetPrincipalIDs, replacement.Actions)
	return nil
}

func (s *GrantService) RevokePrincipalResource(
	ctx context.Context,
	issuer domain.Principal,
	targetPrincipalID string,
	resource domain.Resource,
	policyContext domain.EvaluationContext,
) error {
	issuer.ID = strings.TrimSpace(issuer.ID)
	targetPrincipalID = strings.TrimSpace(targetPrincipalID)
	resource.Type = strings.TrimSpace(resource.Type)
	resource.ID = strings.TrimSpace(resource.ID)
	if issuer.Type != domain.PrincipalTypeUser || issuer.ID == "" || targetPrincipalID == "" ||
		resource.Type == "" || resource.ID == "" {
		return domain.ErrInvalidGrant
	}
	if err := s.requireManager(issuer, []string{targetPrincipalID}, resource, policyContext); err != nil {
		return err
	}
	if err := s.repo.Revoke(ctx, domain.GrantQuery{
		PrincipalUserIDs: []string{targetPrincipalID}, Resource: resource,
	}, s.now()); err != nil {
		return err
	}
	s.logMutation(ctx, "revoke_principal_resource", issuer.ID, resource, []string{targetPrincipalID}, nil)
	return nil
}

// RevokeResource is reserved for domain lifecycle cleanup after a resource is deleted.
func (s *GrantService) RevokeResource(ctx context.Context, resource domain.Resource) error {
	if strings.TrimSpace(resource.Type) == "" || strings.TrimSpace(resource.ID) == "" {
		return domain.ErrInvalidGrant
	}
	return s.repo.Revoke(ctx, domain.GrantQuery{Resource: resource}, s.now())
}

func (s *GrantService) validateManagement(command ReplaceGrantsCommand) error {
	if err := validateGrantCommandShape(command); err != nil {
		return err
	}
	if err := validateConditions(command.Conditions); err != nil {
		return err
	}
	return s.validateGrantActions(command.Actions, command.Resource.Type)
}

func validateGrantCommandShape(command ReplaceGrantsCommand) error {
	if command.Issuer.Type != domain.PrincipalTypeUser || strings.TrimSpace(command.Issuer.ID) == "" {
		return domain.ErrInvalidGrant
	}
	if len(command.TargetPrincipalIDs) == 0 {
		return domain.ErrInvalidGrant
	}
	if strings.TrimSpace(command.Resource.Type) == "" || strings.TrimSpace(command.Resource.ID) == "" {
		return domain.ErrInvalidGrant
	}
	if command.Effect != domain.EffectAllow && command.Effect != domain.EffectDeny {
		return domain.ErrInvalidGrant
	}
	if command.ValidFrom != nil && command.ExpiresAt != nil && *command.ExpiresAt <= *command.ValidFrom {
		return domain.ErrInvalidGrant
	}
	return nil
}

func (s *GrantService) validateGrantActions(actions []string, resourceType string) error {
	for _, actionName := range actions {
		action, ok := s.registry.Action(actionName)
		if !ok || !action.Grantable || action.ResourceType != resourceType {
			return fmt.Errorf("%w: %s", domain.ErrInvalidGrant, actionName)
		}
	}
	return nil
}

func normalizeReplaceCommand(command ReplaceGrantsCommand) ReplaceGrantsCommand {
	command.Issuer.ID = strings.TrimSpace(command.Issuer.ID)
	command.TargetPrincipalIDs = uniqueNonEmpty(command.TargetPrincipalIDs)
	command.Actions = uniqueNonEmpty(command.Actions)
	command.Resource.Type = strings.TrimSpace(command.Resource.Type)
	command.Resource.ID = strings.TrimSpace(command.Resource.ID)
	return command
}

func (s *GrantService) requireManager(
	issuer domain.Principal,
	targetPrincipalIDs []string,
	resource domain.Resource,
	policyContext domain.EvaluationContext,
) error {
	provider, ok := s.registry.Provider(resource.Type)
	if !ok {
		return domain.ErrInvalidGrant
	}
	if err := s.requireBoundaryPermissions(issuer, resource.Type); err != nil {
		return err
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

// requireBoundaryPermissions denies grant management unless the issuer holds the
// global boundary permission for every action the registry marks grantable for
// resourceType, independent of any HTTP route level permission check. It covers
// Grant, Replace, ReplaceMany, Revoke, and RevokePrincipalResource uniformly,
// including RevokePrincipalResource, which has no per-call action list of its own.
func (s *GrantService) requireBoundaryPermissions(issuer domain.Principal, resourceType string) error {
	for _, actionName := range s.registry.GrantableActions(resourceType) {
		action, ok := s.registry.Action(actionName)
		if !ok {
			return domain.ErrInvalidGrant
		}
		if _, held := issuer.GlobalPermissions[action.BoundaryPermission]; !held {
			return domain.ErrGrantManagementForbidden
		}
	}
	return nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func (s *GrantService) logMutation(
	ctx context.Context,
	operation, issuerID string,
	resource domain.Resource,
	principalIDs, actions []string,
) {
	logger.FromContext(ctx).Info("authorization grants changed",
		zap.String("component", "authorization"),
		zap.String("operation", operation),
		zap.String("issuer_principal_id", issuerID),
		zap.String("resource_type", resource.Type),
		zap.String("resource_id", resource.ID),
		zap.Strings("target_principal_ids", principalIDs),
		zap.Strings("actions", actions),
	)
}
