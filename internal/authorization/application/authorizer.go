package application

import (
	"context"
	"sort"
	"strings"

	"go.uber.org/zap"

	"mycourse-io-be/internal/authorization/domain"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/timex"
)

type Authorizer struct {
	registry *Registry
	grants   domain.GrantRepository
	now      func() int64
}

type preparedAuthorization struct {
	request          domain.AuthorizationRequest
	providerDecision domain.PolicyDecision
}

func NewAuthorizer(registry *Registry, grants domain.GrantRepository) *Authorizer {
	return &Authorizer{registry: registry, grants: grants, now: timex.NowUnix}
}

func (a *Authorizer) Authorize(ctx context.Context, request domain.AuthorizationRequest) (domain.Decision, error) {
	decision, err := a.authorize(ctx, request)
	a.record(ctx, request, decision, err)
	return decision, err
}

func (a *Authorizer) authorize(ctx context.Context, request domain.AuthorizationRequest) (domain.Decision, error) {
	prepared, earlyDecision, stop, err := a.prepare(request)
	if stop {
		return earlyDecision, err
	}
	now := a.now()
	grants, err := a.grants.ListActive(ctx, domain.GrantQuery{
		PrincipalUserIDs: []string{prepared.request.Principal.ID},
		ActionNames:      []string{prepared.request.Action},
		Resource:         prepared.request.Resource,
		Effects:          []domain.Effect{domain.EffectAllow, domain.EffectDeny},
	}, now)
	if err != nil {
		return deny(domain.ReasonEvaluationError), err
	}
	return decidePreparedAuthorization(prepared, grants, now)
}

func (a *Authorizer) prepare(request domain.AuthorizationRequest) (preparedAuthorization, domain.Decision, bool, error) {
	normalizeAuthorizationRequest(&request)
	provider, earlyDecision, stop := a.preflight(request)
	if stop {
		return preparedAuthorization{}, earlyDecision, true, nil
	}
	providerDecision, err := provider.Evaluate(request)
	if err != nil {
		return preparedAuthorization{}, deny(domain.ReasonEvaluationError), true, err
	}
	if providerDecision.Effect == domain.PolicyDeny {
		reason := providerDecision.Reason
		if reason == "" {
			reason = domain.ReasonProviderDeny
		}
		return preparedAuthorization{}, deny(reason), true, nil
	}
	return preparedAuthorization{request: request, providerDecision: providerDecision}, domain.Decision{}, false, nil
}

func decidePreparedAuthorization(
	prepared preparedAuthorization,
	grants []domain.Grant,
	now int64,
) (domain.Decision, error) {
	allowIDs, denyIDs, err := matchingGrantIDs(grants, prepared.request.Context.Attributes, now)
	if err != nil {
		return deny(domain.ReasonEvaluationError), err
	}
	if len(denyIDs) > 0 {
		sort.Strings(denyIDs)
		return domain.Decision{Effect: domain.EffectDeny, Reason: domain.ReasonExplicitDeny, MatchedGrantIDs: denyIDs}, nil
	}
	if prepared.providerDecision.Effect == domain.PolicyAllow {
		return domain.Decision{Effect: domain.EffectAllow, Reason: domain.ReasonProviderAllow}, nil
	}
	if len(allowIDs) > 0 {
		sort.Strings(allowIDs)
		return domain.Decision{Effect: domain.EffectAllow, Reason: domain.ReasonGrantAllow, MatchedGrantIDs: allowIDs}, nil
	}
	return deny(domain.ReasonImplicitDeny), nil
}

func normalizeAuthorizationRequest(request *domain.AuthorizationRequest) {
	request.Principal.ID = strings.TrimSpace(request.Principal.ID)
	request.Action = strings.TrimSpace(request.Action)
	request.Resource.Type = strings.TrimSpace(request.Resource.Type)
	request.Resource.ID = strings.TrimSpace(request.Resource.ID)
}

func (a *Authorizer) preflight(request domain.AuthorizationRequest) (domain.PolicyProvider, domain.Decision, bool) {
	if request.Principal.Type != domain.PrincipalTypeUser || request.Principal.ID == "" {
		return nil, deny(domain.ReasonInvalidPrincipal), true
	}
	action, ok := a.registry.Action(request.Action)
	if !ok {
		return nil, deny(domain.ReasonUnknownAction), true
	}
	if request.Resource.Type == "" || request.Resource.ID == "" || action.ResourceType != request.Resource.Type {
		return nil, deny(domain.ReasonResourceTypeMismatch), true
	}
	if _, ok := request.Principal.GlobalPermissions[action.BoundaryPermission]; !ok {
		return nil, deny(domain.ReasonMissingGlobalPermission), true
	}
	provider, ok := a.registry.Provider(request.Resource.Type)
	if !ok {
		return nil, deny(domain.ReasonUnknownAction), true
	}
	return provider, domain.Decision{}, false
}

func matchingGrantIDs(grants []domain.Grant, attributes map[string]any, now int64) ([]string, []string, error) {
	allowIDs := make([]string, 0)
	denyIDs := make([]string, 0)
	for _, grant := range grants {
		if !grant.ActiveAt(now) {
			continue
		}
		matched, conditionErr := conditionsMatch(grant.Conditions, attributes)
		if conditionErr != nil {
			return nil, nil, conditionErr
		}
		if !matched {
			continue
		}
		switch grant.Effect {
		case domain.EffectDeny:
			denyIDs = append(denyIDs, grant.ID)
		case domain.EffectAllow:
			allowIDs = append(allowIDs, grant.ID)
		}
	}
	return allowIDs, denyIDs, nil
}

func deny(reason domain.DecisionReason) domain.Decision {
	return domain.Decision{Effect: domain.EffectDeny, Reason: reason}
}

func (a *Authorizer) record(ctx context.Context, request domain.AuthorizationRequest, decision domain.Decision, err error) {
	fields := []zap.Field{
		zap.String("component", "authorization"),
		zap.String("principal_type", string(request.Principal.Type)),
		zap.String("principal_id", request.Principal.ID),
		zap.String("action", request.Action),
		zap.String("resource_type", request.Resource.Type),
		zap.String("resource_id", request.Resource.ID),
		zap.String("effect", string(decision.Effect)),
		zap.String("reason", string(decision.Reason)),
		zap.Strings("matched_grant_ids", decision.MatchedGrantIDs),
	}
	log := logger.FromContext(ctx)
	if err != nil {
		log.Warn("authorization decision failed closed", append(fields, zap.Error(err))...)
		return
	}
	if decision.Allowed() {
		log.Debug("authorization decision", fields...)
		return
	}
	log.Warn("authorization decision", fields...)
}
