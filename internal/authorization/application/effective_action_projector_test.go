package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"mycourse-io-be/internal/authorization/domain"
)

type testPrincipalResolver struct {
	permissions map[string]map[string]struct{}
	err         error
	calls       int
	requested   []string
}

func (r *testPrincipalResolver) ResolvePrincipals(
	_ context.Context,
	userIDs []string,
) (map[string]domain.Principal, error) {
	r.calls++
	r.requested = append([]string(nil), userIDs...)
	if r.err != nil {
		return nil, r.err
	}
	principals := make(map[string]domain.Principal, len(userIDs))
	for _, userID := range userIDs {
		principals[userID] = domain.Principal{
			Type:              domain.PrincipalTypeUser,
			ID:                userID,
			GlobalPermissions: r.permissions[userID],
		}
	}
	return principals, nil
}

type effectiveProjectorFixture struct {
	repo       *memoryGrantRepository
	resolver   *testPrincipalResolver
	authorizer *Authorizer
	projector  *EffectiveActionProjector
	subjects   []EffectiveActionSubject
}

func projectionTestGrants(now int64) []domain.Grant {
	return []domain.Grant{
		{
			ID: "u1-update", PrincipalUserID: "u1", ActionName: testAction,
			ResourceType: testResource, ResourceID: "doc1", Effect: domain.EffectAllow,
			Conditions: []domain.Condition{{
				Operator: domain.ConditionStringEquals, Key: "tenant", Values: []any{"alpha"},
			}},
		},
		{
			ID: "u1-read-expired", PrincipalUserID: "u1", ActionName: testActionRead,
			ResourceType: testResource, ResourceID: "doc1", Effect: domain.EffectAllow,
			ExpiresAt: int64Pointer(now),
		},
		{
			ID: "u2-update", PrincipalUserID: "u2", ActionName: testAction,
			ResourceType: testResource, ResourceID: "doc1", Effect: domain.EffectAllow,
		},
		{
			ID: "owner-update-deny", PrincipalUserID: "owner", ActionName: testAction,
			ResourceType: testResource, ResourceID: "doc1", Effect: domain.EffectDeny,
		},
		{
			ID: "blocked-read", PrincipalUserID: "blocked", ActionName: testActionRead,
			ResourceType: testResource, ResourceID: "doc1", Effect: domain.EffectAllow,
		},
	}
}

func newEffectiveProjectorFixture(t *testing.T) effectiveProjectorFixture {
	t.Helper()
	const now = int64(1_700_000_000)
	repo := &memoryGrantRepository{grants: projectionTestGrants(now)}
	resolver := &testPrincipalResolver{permissions: map[string]map[string]struct{}{
		"u1":      {testBoundary: {}},
		"u2":      {},
		"owner":   {testBoundary: {}},
		"blocked": {testBoundary: {}},
	}}
	authorizer := NewAuthorizer(testRegistry(t), repo)
	authorizer.now = func() int64 { return now }
	return effectiveProjectorFixture{
		repo: repo, resolver: resolver, authorizer: authorizer,
		projector: NewEffectiveActionProjector(authorizer, resolver),
		subjects: []EffectiveActionSubject{
			{UserID: "u2", Context: domain.EvaluationContext{DomainFacts: testPolicyFacts{}}},
			{UserID: "owner", Context: domain.EvaluationContext{DomainFacts: testPolicyFacts{Allow: true}}},
			{UserID: "u1", Context: domain.EvaluationContext{
				DomainFacts: testPolicyFacts{}, Attributes: map[string]any{"tenant": "alpha"},
			}},
			{UserID: "blocked", Context: domain.EvaluationContext{DomainFacts: testPolicyFacts{Deny: true}}},
		},
	}
}

func TestEffectiveActionProjectorMatchesAuthorizerInOneBatch(t *testing.T) {
	fixture := newEffectiveProjectorFixture(t)

	got, err := fixture.projector.Project(context.Background(), EffectiveActionsRequest{
		Resource:    domain.Resource{Type: testResource, ID: "doc1"},
		ActionNames: []string{testAction, testActionRead, testAction},
		Subjects:    fixture.subjects,
	})
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	want := map[string][]string{
		"u1":      {testAction},
		"u2":      {},
		"owner":   {testActionRead},
		"blocked": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Project = %#v, want %#v", got, want)
	}
	if fixture.resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", fixture.resolver.calls)
	}
	if fixture.repo.listCalls != 1 {
		t.Fatalf("grant repository calls = %d, want 1", fixture.repo.listCalls)
	}
	if wantIDs := []string{"blocked", "owner", "u1", "u2"}; !reflect.DeepEqual(fixture.resolver.requested, wantIDs) {
		t.Fatalf("resolved IDs = %#v, want %#v", fixture.resolver.requested, wantIDs)
	}
	assertProjectionMatchesAuthorize(t, fixture, got)
}

func assertProjectionMatchesAuthorize(
	t *testing.T,
	fixture effectiveProjectorFixture,
	projected map[string][]string,
) {
	t.Helper()
	for _, subject := range fixture.subjects {
		for _, action := range []string{testAction, testActionRead} {
			decision, err := fixture.authorizer.Authorize(context.Background(), domain.AuthorizationRequest{
				Principal: resolverPrincipal(fixture.resolver, subject.UserID), Action: action,
				Resource: domain.Resource{Type: testResource, ID: "doc1"}, Context: subject.Context,
			})
			if err != nil {
				t.Fatalf("Authorize %s/%s: %v", subject.UserID, action, err)
			}
			if containsString(projected[subject.UserID], action) != decision.Allowed() {
				t.Fatalf("projection parity failed for %s/%s: actions=%v decision=%#v", subject.UserID, action, projected[subject.UserID], decision)
			}
		}
	}
}

func TestEffectiveActionProjectorFailsClosed(t *testing.T) {
	request := EffectiveActionsRequest{
		Resource:    domain.Resource{Type: testResource, ID: "doc1"},
		ActionNames: []string{testAction},
		Subjects: []EffectiveActionSubject{{
			UserID: "u1", Context: domain.EvaluationContext{DomainFacts: testPolicyFacts{}},
		}},
	}
	for _, tt := range []struct {
		name        string
		resolverErr error
		repoErr     error
		facts       testPolicyFacts
	}{
		{name: "principal resolver", resolverErr: errors.New("resolver failed")},
		{name: "grant repository", repoErr: errors.New("repository failed")},
		{name: "policy provider", facts: testPolicyFacts{Fail: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &memoryGrantRepository{listErr: tt.repoErr}
			resolver := &testPrincipalResolver{
				permissions: map[string]map[string]struct{}{"u1": {testBoundary: {}}},
				err:         tt.resolverErr,
			}
			authorizer := NewAuthorizer(testRegistry(t), repo)
			projector := NewEffectiveActionProjector(authorizer, resolver)
			request.Subjects[0].Context.DomainFacts = tt.facts
			got, err := projector.Project(context.Background(), request)
			if err == nil {
				t.Fatalf("Project = %#v, nil; want fail-closed error", got)
			}
			if got != nil {
				t.Fatalf("partial projection = %#v, want nil", got)
			}
		})
	}
}

func TestEffectiveActionProjectorReturnsStableEmptyEntries(t *testing.T) {
	resolver := &testPrincipalResolver{permissions: map[string]map[string]struct{}{}}
	projector := NewEffectiveActionProjector(NewAuthorizer(testRegistry(t), &memoryGrantRepository{}), resolver)

	got, err := projector.Project(context.Background(), EffectiveActionsRequest{
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
		Subjects: []EffectiveActionSubject{{UserID: " u1 "}, {UserID: "u1"}},
	})
	if err != nil {
		t.Fatalf("Project empty actions: %v", err)
	}
	if want := map[string][]string{"u1": {}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("empty projection = %#v, want %#v", got, want)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls for empty action set = %d, want 0", resolver.calls)
	}
}

func resolverPrincipal(resolver *testPrincipalResolver, userID string) domain.Principal {
	return domain.Principal{
		Type:              domain.PrincipalTypeUser,
		ID:                userID,
		GlobalPermissions: resolver.permissions[userID],
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
