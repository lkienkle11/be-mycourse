package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"mycourse-io-be/internal/authorization/domain"
)

const (
	testAction     = "document:update"
	testActionRead = "document:read_scoped"
	testResource   = "document"
	testBoundary   = "document:update_global"
)

type testPolicyFacts struct {
	Allow     bool
	Deny      bool
	ManagerID string
	Fail      bool
}

type testPolicyProvider struct{}

type duplicateActionProvider struct{}

func (duplicateActionProvider) ResourceType() string { return "other" }
func (duplicateActionProvider) Actions() []domain.ActionDefinition {
	return []domain.ActionDefinition{{Name: testAction, ResourceType: "other", BoundaryPermission: testBoundary}}
}
func (duplicateActionProvider) Evaluate(domain.AuthorizationRequest) (domain.PolicyDecision, error) {
	return domain.PolicyDecision{}, nil
}
func (duplicateActionProvider) CanManageGrants(domain.GrantManagementRequest) (bool, error) {
	return false, nil
}

func (testPolicyProvider) ResourceType() string { return testResource }

func (testPolicyProvider) Actions() []domain.ActionDefinition {
	return []domain.ActionDefinition{
		{Name: testAction, ResourceType: testResource, BoundaryPermission: testBoundary, Grantable: true},
		{Name: testActionRead, ResourceType: testResource, BoundaryPermission: testBoundary, Grantable: true},
	}
}

func (testPolicyProvider) Evaluate(request domain.AuthorizationRequest) (domain.PolicyDecision, error) {
	facts, ok := request.Context.DomainFacts.(testPolicyFacts)
	if !ok {
		return domain.PolicyDecision{}, domain.ErrInvalidPolicyContext
	}
	if facts.Fail {
		return domain.PolicyDecision{}, errors.New("provider failed")
	}
	if facts.Deny {
		return domain.PolicyDecision{Effect: domain.PolicyDeny, Reason: domain.ReasonProviderDeny}, nil
	}
	if facts.Allow {
		return domain.PolicyDecision{Effect: domain.PolicyAllow, Reason: domain.ReasonProviderAllow}, nil
	}
	return domain.PolicyDecision{Effect: domain.PolicyNeutral}, nil
}

func (testPolicyProvider) CanManageGrants(request domain.GrantManagementRequest) (bool, error) {
	facts, ok := request.Context.DomainFacts.(testPolicyFacts)
	if !ok {
		return false, domain.ErrInvalidPolicyContext
	}
	if facts.Fail {
		return false, errors.New("provider failed")
	}
	return request.Issuer.ID == facts.ManagerID, nil
}

type memoryGrantRepository struct {
	actions   []domain.ActionDefinition
	grants    []domain.Grant
	nextID    int
	listCalls int
	listErr   error
}

func (r *memoryGrantRepository) UpsertActions(_ context.Context, actions []domain.ActionDefinition) error {
	r.actions = append([]domain.ActionDefinition(nil), actions...)
	return nil
}

func (r *memoryGrantRepository) ListActive(_ context.Context, query domain.GrantQuery, now int64) ([]domain.Grant, error) {
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]domain.Grant, 0)
	for _, grant := range r.grants {
		if !grant.ActiveAt(now) || !queryMatchesGrant(query, grant) {
			continue
		}
		out = append(out, grant)
	}
	return out, nil
}

func (r *memoryGrantRepository) Create(_ context.Context, grant *domain.Grant) error {
	r.nextID++
	copy := *grant
	if copy.ID == "" {
		copy.ID = fmt.Sprintf("g%d", r.nextID)
	}
	grant.ID = copy.ID
	r.grants = append(r.grants, copy)
	return nil
}

func (r *memoryGrantRepository) Revoke(_ context.Context, query domain.GrantQuery, revokedAt int64) error {
	for i := range r.grants {
		if r.grants[i].RevokedAt == nil && queryMatchesGrant(query, r.grants[i]) {
			r.grants[i].RevokedAt = &revokedAt
		}
	}
	return nil
}

func (r *memoryGrantRepository) ReplaceMany(_ context.Context, replacement domain.GrantReplacement, now int64) error {
	_ = r.Revoke(context.Background(), domain.GrantQuery{
		PrincipalUserIDs: replacement.PrincipalUserIDs,
		ActionNames:      replacement.ManagedActions,
		Resource:         replacement.Resource,
		Effects:          []domain.Effect{replacement.Effect},
	}, now)
	for _, principalID := range replacement.PrincipalUserIDs {
		for _, action := range replacement.Actions {
			r.nextID++
			r.grants = append(r.grants, domain.Grant{
				ID: fmt.Sprintf("g%d", r.nextID), PrincipalUserID: principalID,
				ActionName: action, ResourceType: replacement.Resource.Type,
				ResourceID: replacement.Resource.ID, Effect: replacement.Effect,
				Conditions: replacement.Conditions, ValidFrom: replacement.ValidFrom,
				ExpiresAt: replacement.ExpiresAt, GrantedByUserID: replacement.GrantedByUserID,
				CreatedAt: now,
			})
		}
	}
	return nil
}

func queryMatchesGrant(query domain.GrantQuery, grant domain.Grant) bool {
	return containsOrEmpty(query.PrincipalUserIDs, grant.PrincipalUserID) &&
		containsOrEmpty(query.ActionNames, grant.ActionName) &&
		(query.Resource.Type == "" || query.Resource.Type == grant.ResourceType) &&
		(query.Resource.ID == "" || query.Resource.ID == grant.ResourceID) &&
		containsEffectOrEmpty(query.Effects, grant.Effect)
}

func containsOrEmpty(values []string, wanted string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsEffectOrEmpty(values []domain.Effect, wanted domain.Effect) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	registry, err := NewRegistry(testPolicyProvider{})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return registry
}

func testRequest(facts testPolicyFacts) domain.AuthorizationRequest {
	return domain.AuthorizationRequest{
		Principal: domain.Principal{
			Type: domain.PrincipalTypeUser, ID: "u1",
			GlobalPermissions: map[string]struct{}{testBoundary: {}},
		},
		Action:   testAction,
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
		Context: domain.EvaluationContext{DomainFacts: facts, Attributes: map[string]any{
			"tenant": "alpha", "enabled": true, "score": 7, "request.time": int64(1_700_000_000),
		}},
	}
}

type authorizerDecisionCase struct {
	name       string
	request    domain.AuthorizationRequest
	grants     []domain.Grant
	wantEffect domain.Effect
	wantReason domain.DecisionReason
	wantErr    bool
	repoErr    error
}

func authorizerDecisionCases(now int64) []authorizerDecisionCase {
	allowGrant := domain.Grant{
		ID: "allow", PrincipalUserID: "u1", ActionName: testAction,
		ResourceType: testResource, ResourceID: "doc1", Effect: domain.EffectAllow,
	}
	denyGrant := allowGrant
	denyGrant.ID = "deny"
	denyGrant.Effect = domain.EffectDeny
	expired := allowGrant
	expired.ID = "expired"
	expired.ExpiresAt = int64Pointer(now)
	future := allowGrant
	future.ID = "future"
	future.ValidFrom = int64Pointer(now + 1)
	conditioned := allowGrant
	conditioned.ID = "conditioned"
	conditioned.Conditions = []domain.Condition{{
		Operator: domain.ConditionStringEquals, Key: "tenant", Values: []any{"alpha"},
	}}
	missingBoundary := testRequest(testPolicyFacts{Allow: true})
	missingBoundary.Principal.GlobalPermissions = map[string]struct{}{}
	unknownAction := testRequest(testPolicyFacts{Allow: true})
	unknownAction.Action = "unknown:update"
	resourceMismatch := testRequest(testPolicyFacts{Allow: true})
	resourceMismatch.Resource.Type = "other"
	invalidPrincipal := testRequest(testPolicyFacts{Allow: true})
	invalidPrincipal.Principal.ID = ""
	return []authorizerDecisionCase{
		{name: "provider allow", request: testRequest(testPolicyFacts{Allow: true}), wantEffect: domain.EffectAllow, wantReason: domain.ReasonProviderAllow},
		{name: "grant allow", request: testRequest(testPolicyFacts{}), grants: []domain.Grant{allowGrant}, wantEffect: domain.EffectAllow, wantReason: domain.ReasonGrantAllow},
		{name: "matching condition", request: testRequest(testPolicyFacts{}), grants: []domain.Grant{conditioned}, wantEffect: domain.EffectAllow, wantReason: domain.ReasonGrantAllow},
		{name: "explicit deny wins", request: testRequest(testPolicyFacts{Allow: true}), grants: []domain.Grant{allowGrant, denyGrant}, wantEffect: domain.EffectDeny, wantReason: domain.ReasonExplicitDeny},
		{name: "provider hard deny", request: testRequest(testPolicyFacts{Deny: true}), grants: []domain.Grant{allowGrant}, wantEffect: domain.EffectDeny, wantReason: domain.ReasonProviderDeny},
		{name: "expired grant", request: testRequest(testPolicyFacts{}), grants: []domain.Grant{expired}, wantEffect: domain.EffectDeny, wantReason: domain.ReasonImplicitDeny},
		{name: "future grant", request: testRequest(testPolicyFacts{}), grants: []domain.Grant{future}, wantEffect: domain.EffectDeny, wantReason: domain.ReasonImplicitDeny},
		{name: "implicit deny", request: testRequest(testPolicyFacts{}), wantEffect: domain.EffectDeny, wantReason: domain.ReasonImplicitDeny},
		{name: "provider failure", request: testRequest(testPolicyFacts{Fail: true}), wantEffect: domain.EffectDeny, wantReason: domain.ReasonEvaluationError, wantErr: true},
		{name: "repository failure", request: testRequest(testPolicyFacts{}), repoErr: errors.New("repository failed"), wantEffect: domain.EffectDeny, wantReason: domain.ReasonEvaluationError, wantErr: true},
		{name: "global boundary", request: missingBoundary, wantEffect: domain.EffectDeny, wantReason: domain.ReasonMissingGlobalPermission},
		{name: "unknown action", request: unknownAction, wantEffect: domain.EffectDeny, wantReason: domain.ReasonUnknownAction},
		{name: "resource mismatch", request: resourceMismatch, wantEffect: domain.EffectDeny, wantReason: domain.ReasonResourceTypeMismatch},
		{name: "invalid principal", request: invalidPrincipal, wantEffect: domain.EffectDeny, wantReason: domain.ReasonInvalidPrincipal},
	}
}

func TestAuthorizerDecisionPrecedence(t *testing.T) {
	const now = int64(1_700_000_000)
	for _, tt := range authorizerDecisionCases(now) {
		t.Run(tt.name, func(t *testing.T) {
			repo := &memoryGrantRepository{grants: tt.grants, listErr: tt.repoErr}
			authorizer := NewAuthorizer(testRegistry(t), repo)
			authorizer.now = func() int64 { return now }
			decision, err := authorizer.Authorize(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if decision.Effect != tt.wantEffect || decision.Reason != tt.wantReason {
				t.Fatalf("decision = %#v, want effect=%s reason=%s", decision, tt.wantEffect, tt.wantReason)
			}
		})
	}
}

func TestConditionOperators(t *testing.T) {
	attributes := map[string]any{
		"name": "alpha", "enabled": true, "score": float64(7), "request.time": int64(1_700_000_000),
	}
	conditions := []domain.Condition{
		{Operator: domain.ConditionStringEquals, Key: "name", Values: []any{"beta", "alpha"}},
		{Operator: domain.ConditionBoolEquals, Key: "enabled", Values: []any{true}},
		{Operator: domain.ConditionNumericEquals, Key: "score", Values: []any{7}},
		{Operator: domain.ConditionDateAfter, Key: "request.time", Values: []any{int64(1_600_000_000)}},
		{Operator: domain.ConditionDateBefore, Key: "request.time", Values: []any{int64(1_800_000_000)}},
	}
	matched, err := conditionsMatch(conditions, attributes)
	if err != nil || !matched {
		t.Fatalf("conditionsMatch = %v, %v; want true, nil", matched, err)
	}
	conditions[0].Key = "missing"
	matched, err = conditionsMatch(conditions, attributes)
	if err != nil || matched {
		t.Fatalf("missing attribute = %v, %v; want false, nil", matched, err)
	}
	for _, invalid := range []domain.Condition{
		{Operator: "Unknown", Key: "name", Values: []any{"alpha"}},
		{Operator: domain.ConditionBoolEquals, Key: "name", Values: []any{true}},
	} {
		matched, err = conditionsMatch([]domain.Condition{invalid}, attributes)
		if err != nil || matched {
			t.Fatalf("invalid condition = %v, %v; want false, nil", matched, err)
		}
	}
}

type grantServiceFixture struct {
	repo          *memoryGrantRepository
	service       *GrantService
	issuer        domain.Principal
	resource      domain.Resource
	policyContext domain.EvaluationContext
}

func newGrantServiceFixture(t *testing.T) grantServiceFixture {
	t.Helper()
	const now = int64(1_700_000_000)
	repo := &memoryGrantRepository{}
	service := NewGrantService(testRegistry(t), repo)
	service.now = func() int64 { return now }
	if err := service.SyncActions(context.Background()); err != nil {
		t.Fatalf("SyncActions: %v", err)
	}
	return grantServiceFixture{
		repo: repo, service: service,
		issuer: domain.Principal{
			Type: domain.PrincipalTypeUser, ID: "owner",
			GlobalPermissions: map[string]struct{}{testBoundary: {}},
		},
		resource:      domain.Resource{Type: testResource, ID: "doc1"},
		policyContext: domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}},
	}
}

func (f grantServiceFixture) replaceBoth(t *testing.T) {
	t.Helper()
	if err := f.service.ReplaceMany(context.Background(), ReplaceGrantsCommand{
		Issuer: f.issuer, TargetPrincipalIDs: []string{"u2", "u1", "u1"},
		Actions: []string{testActionRead, testAction, testAction}, Resource: f.resource,
		Effect: domain.EffectAllow, Context: f.policyContext,
	}); err != nil {
		t.Fatalf("ReplaceMany: %v", err)
	}
}

func TestGrantServiceReplaceManyCreatesManagedStatements(t *testing.T) {
	f := newGrantServiceFixture(t)
	if len(f.repo.actions) != 2 {
		t.Fatalf("synced actions = %d, want 2", len(f.repo.actions))
	}
	f.replaceBoth(t)
	wantActions := []string{testActionRead, testAction}
	for _, userID := range []string{"u1", "u2"} {
		gotActions := activeActionsForPrincipal(f.repo.grants, userID, 1_700_000_000)
		if !reflect.DeepEqual(gotActions, wantActions) {
			t.Fatalf("active actions for %s = %#v, want %#v", userID, gotActions, wantActions)
		}
	}
}

func TestGrantServiceAppendsIndependentStatements(t *testing.T) {
	const now = int64(1_700_000_000)
	f := newGrantServiceFixture(t)
	commands := additiveGrantCommands(f, now)
	for i, command := range commands {
		grant, err := f.service.Grant(context.Background(), command)
		if err != nil {
			t.Fatalf("Grant statement %d: %v", i, err)
		}
		if grant.ID == "" {
			t.Fatalf("Grant statement %d has empty ID", i)
		}
	}
	if len(f.repo.grants) != len(commands) {
		t.Fatalf("stored grants = %d, want %d additive statements", len(f.repo.grants), len(commands))
	}
	if f.repo.grants[0].RevokedAt != nil {
		t.Fatal("expired statement must remain unrevoked history")
	}

	authorizer := NewAuthorizer(testRegistry(t), f.repo)
	authorizer.now = func() int64 { return now }
	request := testRequest(testPolicyFacts{})
	decision, err := authorizer.Authorize(context.Background(), request)
	if err != nil {
		t.Fatalf("Authorize alpha: %v", err)
	}
	if decision.Effect != domain.EffectAllow || decision.Reason != domain.ReasonGrantAllow {
		t.Fatalf("alpha decision = %#v, want grant allow", decision)
	}
	if want := []string{"g2", "g4"}; !reflect.DeepEqual(decision.MatchedGrantIDs, want) {
		t.Fatalf("alpha matched IDs = %#v, want %#v", decision.MatchedGrantIDs, want)
	}

	request.Context.Attributes["tenant"] = "beta"
	decision, err = authorizer.Authorize(context.Background(), request)
	if err != nil {
		t.Fatalf("Authorize beta: %v", err)
	}
	if want := []string{"g3"}; !reflect.DeepEqual(decision.MatchedGrantIDs, want) {
		t.Fatalf("beta matched IDs = %#v, want %#v", decision.MatchedGrantIDs, want)
	}
}

func additiveGrantCommands(f grantServiceFixture, now int64) []GrantCommand {
	alphaCondition := []domain.Condition{{
		Operator: domain.ConditionStringEquals,
		Key:      "tenant",
		Values:   []any{"alpha"},
	}}
	betaCondition := []domain.Condition{{
		Operator: domain.ConditionStringEquals,
		Key:      "tenant",
		Values:   []any{"beta"},
	}}
	expiredAt := now - 1

	return []GrantCommand{
		{
			Issuer: f.issuer, TargetPrincipalID: "u1", Action: testAction,
			Resource: f.resource, Effect: domain.EffectAllow, Conditions: alphaCondition,
			ExpiresAt: &expiredAt, Context: f.policyContext,
		},
		{
			Issuer: f.issuer, TargetPrincipalID: "u1", Action: testAction,
			Resource: f.resource, Effect: domain.EffectAllow, Conditions: alphaCondition,
			Context: f.policyContext,
		},
		{
			Issuer: f.issuer, TargetPrincipalID: "u1", Action: testAction,
			Resource: f.resource, Effect: domain.EffectAllow, Conditions: betaCondition,
			Context: f.policyContext,
		},
		{
			Issuer: f.issuer, TargetPrincipalID: "u1", Action: testAction,
			Resource: f.resource, Effect: domain.EffectAllow, Conditions: alphaCondition,
			Context: f.policyContext,
		},
	}
}

func TestGrantServiceRevokeReaddAndClear(t *testing.T) {
	const now = int64(1_700_000_000)
	f := newGrantServiceFixture(t)
	f.replaceBoth(t)
	if err := f.service.RevokePrincipalResource(context.Background(), f.issuer, "u1", f.resource, f.policyContext); err != nil {
		t.Fatalf("RevokePrincipalResource: %v", err)
	}
	wantActions := []string{testActionRead, testAction}
	if got := activeActionsForPrincipal(f.repo.grants, "u1", now); len(got) != 0 {
		t.Fatalf("u1 actions after revoke = %#v, want empty", got)
	}
	if got := activeActionsForPrincipal(f.repo.grants, "u2", now); !reflect.DeepEqual(got, wantActions) {
		t.Fatalf("u2 actions after u1 revoke = %#v, want %#v", got, wantActions)
	}
	oldGrantCount := len(f.repo.grants)
	if err := f.service.Replace(context.Background(), ReplaceGrantsCommand{
		Issuer: f.issuer, TargetPrincipalIDs: []string{" u1 "}, Actions: []string{" " + testAction + " "},
		Resource: f.resource, Effect: domain.EffectAllow, Context: f.policyContext,
	}); err != nil {
		t.Fatalf("re-add after revoke: %v", err)
	}
	if len(f.repo.grants) != oldGrantCount+1 {
		t.Fatalf("re-add must create grant history row: before=%d after=%d", oldGrantCount, len(f.repo.grants))
	}
	active := 0
	for _, grant := range f.repo.grants {
		if grant.PrincipalUserID == "u1" && grant.ActionName == testAction && grant.ActiveAt(now) {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active re-added grants = %d, want 1", active)
	}
	if err := f.service.Replace(context.Background(), ReplaceGrantsCommand{
		Issuer: f.issuer, TargetPrincipalIDs: []string{"u1"}, Actions: []string{},
		Resource: f.resource, Effect: domain.EffectAllow, Context: f.policyContext,
	}); err != nil {
		t.Fatalf("clear actions: %v", err)
	}
	if got := activeActionsForPrincipal(f.repo.grants, "u1", now); len(got) != 0 {
		t.Fatalf("actions after clear = %#v, want empty", got)
	}
}

func activeActionsForPrincipal(grants []domain.Grant, principalID string, now int64) []string {
	actions := make([]string, 0)
	for _, grant := range grants {
		if grant.PrincipalUserID == principalID && grant.Effect == domain.EffectAllow && grant.ActiveAt(now) {
			actions = append(actions, grant.ActionName)
		}
	}
	sort.Strings(actions)
	return actions
}

func TestGrantServiceRejectsUnauthorizedManager(t *testing.T) {
	f := newGrantServiceFixture(t)
	unauthorized := f.issuer
	unauthorized.ID = "outsider"
	err := f.service.Replace(context.Background(), ReplaceGrantsCommand{
		Issuer: unauthorized, TargetPrincipalIDs: []string{"u1"}, Actions: []string{testAction},
		Resource: f.resource, Effect: domain.EffectAllow, Context: f.policyContext,
	})
	if !errors.Is(err, domain.ErrGrantManagementForbidden) {
		t.Fatalf("unauthorized replace error = %v", err)
	}
}

func TestGrantServiceRequireBoundaryPermission(t *testing.T) {
	f := newGrantServiceFixture(t)

	t.Run("owner with the boundary permission succeeds", func(t *testing.T) {
		if err := f.service.Replace(context.Background(), ReplaceGrantsCommand{
			Issuer: f.issuer, TargetPrincipalIDs: []string{"u1"}, Actions: []string{testAction},
			Resource: f.resource, Effect: domain.EffectAllow, Context: f.policyContext,
		}); err != nil {
			t.Fatalf("owner with boundary permission: %v", err)
		}
	})

	t.Run("owner without the boundary permission is denied", func(t *testing.T) {
		issuerNoBoundary := f.issuer
		issuerNoBoundary.GlobalPermissions = map[string]struct{}{}
		err := f.service.Replace(context.Background(), ReplaceGrantsCommand{
			Issuer: issuerNoBoundary, TargetPrincipalIDs: []string{"u1"}, Actions: []string{testAction},
			Resource: f.resource, Effect: domain.EffectAllow, Context: f.policyContext,
		})
		if !errors.Is(err, domain.ErrGrantManagementForbidden) {
			t.Fatalf("owner without boundary permission error = %v, want ErrGrantManagementForbidden", err)
		}
	})

	t.Run("non owner with the boundary permission is still denied by the resource level check", func(t *testing.T) {
		outsider := domain.Principal{
			Type: domain.PrincipalTypeUser, ID: "outsider",
			GlobalPermissions: map[string]struct{}{testBoundary: {}},
		}
		err := f.service.Replace(context.Background(), ReplaceGrantsCommand{
			Issuer: outsider, TargetPrincipalIDs: []string{"u1"}, Actions: []string{testAction},
			Resource: f.resource, Effect: domain.EffectAllow, Context: f.policyContext,
		})
		if !errors.Is(err, domain.ErrGrantManagementForbidden) {
			t.Fatalf("non owner with boundary permission error = %v, want ErrGrantManagementForbidden", err)
		}
	})

	t.Run("RevokeResource has no issuing principal and stays unaffected", func(t *testing.T) {
		if err := f.service.RevokeResource(context.Background(), f.resource); err != nil {
			t.Fatalf("RevokeResource: %v", err)
		}
	})
}

func TestRegistryRejectsDuplicateProviderAndAction(t *testing.T) {
	if _, err := NewRegistry(testPolicyProvider{}, testPolicyProvider{}); !errors.Is(err, domain.ErrDuplicateProvider) {
		t.Fatalf("duplicate provider error = %v", err)
	}
	actions := testPolicyProvider{}.Actions()
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	if actions[0].Name == actions[1].Name {
		t.Fatal("test provider actions must be unique")
	}
	if _, err := NewRegistry(testPolicyProvider{}, duplicateActionProvider{}); !errors.Is(err, domain.ErrDuplicateAction) {
		t.Fatalf("duplicate action error = %v", err)
	}
}

func int64Pointer(value int64) *int64 { return &value }
