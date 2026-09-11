package application

import (
	"context"
	"sort"
	"strings"

	"mycourse-io-be/internal/authorization/domain"
)

// PrincipalResolver builds trusted principals from server-side identity and RBAC state.
type PrincipalResolver interface {
	ResolvePrincipals(ctx context.Context, userIDs []string) (map[string]domain.Principal, error)
}

type EffectiveActionSubject struct {
	UserID  string
	Context domain.EvaluationContext
}

type EffectiveActionsRequest struct {
	Resource    domain.Resource
	ActionNames []string
	Subjects    []EffectiveActionSubject
}

// EffectiveActionProjector calculates response projections through the same decision path as Authorizer.
type EffectiveActionProjector struct {
	authorizer *Authorizer
	resolver   PrincipalResolver
}

type pendingEffectiveAction struct {
	userID   string
	action   string
	prepared preparedAuthorization
}

func NewEffectiveActionProjector(authorizer *Authorizer, resolver PrincipalResolver) *EffectiveActionProjector {
	return &EffectiveActionProjector{authorizer: authorizer, resolver: resolver}
}

func (p *EffectiveActionProjector) Project(
	ctx context.Context,
	request EffectiveActionsRequest,
) (map[string][]string, error) {
	request.Resource.Type = strings.TrimSpace(request.Resource.Type)
	request.Resource.ID = strings.TrimSpace(request.Resource.ID)
	actions := uniqueNonEmpty(request.ActionNames)
	subjects, userIDs := normalizeEffectiveActionSubjects(request.Subjects)
	result := emptyEffectiveActions(userIDs)
	if len(userIDs) == 0 || len(actions) == 0 {
		return result, nil
	}

	principals, err := p.resolver.ResolvePrincipals(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	pending, err := p.prepareEffectiveActions(request.Resource, actions, subjects, principals)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return result, nil
	}

	now := p.authorizer.now()
	grants, err := p.authorizer.grants.ListActive(ctx, domain.GrantQuery{
		PrincipalUserIDs: userIDs,
		ActionNames:      actions,
		Resource:         request.Resource,
		Effects:          []domain.Effect{domain.EffectAllow, domain.EffectDeny},
	}, now)
	if err != nil {
		return nil, err
	}
	if err := applyEffectiveActionDecisions(result, pending, grants, now); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *EffectiveActionProjector) prepareEffectiveActions(
	resource domain.Resource,
	actions []string,
	subjects []EffectiveActionSubject,
	principals map[string]domain.Principal,
) ([]pendingEffectiveAction, error) {
	pending := make([]pendingEffectiveAction, 0, len(subjects)*len(actions))
	for _, subject := range subjects {
		principal := principals[subject.UserID]
		for _, action := range actions {
			prepared, _, stop, err := p.authorizer.prepare(domain.AuthorizationRequest{
				Principal: principal,
				Action:    action,
				Resource:  resource,
				Context:   subject.Context,
			})
			if err != nil {
				return nil, err
			}
			if stop {
				continue
			}
			pending = append(pending, pendingEffectiveAction{
				userID: subject.UserID, action: action, prepared: prepared,
			})
		}
	}
	return pending, nil
}

func emptyEffectiveActions(userIDs []string) map[string][]string {
	result := make(map[string][]string, len(userIDs))
	for _, userID := range userIDs {
		result[userID] = []string{}
	}
	return result
}

func applyEffectiveActionDecisions(
	result map[string][]string,
	pending []pendingEffectiveAction,
	grants []domain.Grant,
	now int64,
) error {
	grantsByDecision := groupGrantsByDecision(grants)
	for _, item := range pending {
		decision, decisionErr := decidePreparedAuthorization(
			item.prepared,
			grantsByDecision[decisionKey(item.userID, item.action)],
			now,
		)
		if decisionErr != nil {
			return decisionErr
		}
		if decision.Allowed() {
			result[item.userID] = append(result[item.userID], item.action)
		}
	}
	for userID := range result {
		sort.Strings(result[userID])
	}
	return nil
}

func normalizeEffectiveActionSubjects(subjects []EffectiveActionSubject) ([]EffectiveActionSubject, []string) {
	seen := make(map[string]struct{}, len(subjects))
	normalized := make([]EffectiveActionSubject, 0, len(subjects))
	for _, subject := range subjects {
		subject.UserID = strings.TrimSpace(subject.UserID)
		if subject.UserID == "" {
			continue
		}
		if _, exists := seen[subject.UserID]; exists {
			continue
		}
		seen[subject.UserID] = struct{}{}
		normalized = append(normalized, subject)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].UserID < normalized[j].UserID })
	userIDs := make([]string, len(normalized))
	for i := range normalized {
		userIDs[i] = normalized[i].UserID
	}
	return normalized, userIDs
}

func groupGrantsByDecision(grants []domain.Grant) map[string][]domain.Grant {
	result := make(map[string][]domain.Grant)
	for _, grant := range grants {
		key := decisionKey(grant.PrincipalUserID, grant.ActionName)
		result[key] = append(result[key], grant)
	}
	return result
}

func decisionKey(principalID, action string) string {
	return principalID + "\x00" + action
}
