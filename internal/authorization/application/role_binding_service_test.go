package application

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	"mycourse-io-be/internal/authorization/domain"
)

type roleBindingRecord struct {
	principalUserID string
	roleName        string
	resource        domain.Resource
	grantedByUserID string
	createdAt       int64
	revokedAt       *int64
}

type memoryRoleBindingRepository struct {
	bindings      []roleBindingRecord
	assignCalls   int
	assignedBatch []string // principal IDs passed to the last AssignRoleBindings call, for batch-call assertions
}

func (r *memoryRoleBindingRepository) AssignRoleBindings(
	_ context.Context,
	principalUserIDs []string,
	roleName string,
	resource domain.Resource,
	grantedByUserID string,
	now int64,
) error {
	r.assignCalls++
	r.assignedBatch = append([]string(nil), principalUserIDs...)
	for _, principalID := range principalUserIDs {
		r.bindings = append(r.bindings, roleBindingRecord{
			principalUserID: principalID, roleName: roleName, resource: resource,
			grantedByUserID: grantedByUserID, createdAt: now,
		})
	}
	return nil
}

func (r *memoryRoleBindingRepository) RevokeRoleBindings(_ context.Context, query domain.RoleBindingQuery, revokedAt int64) error {
	principalSet := make(map[string]struct{}, len(query.PrincipalUserIDs))
	for _, id := range query.PrincipalUserIDs {
		principalSet[id] = struct{}{}
	}
	for i := range r.bindings {
		b := &r.bindings[i]
		if b.revokedAt != nil {
			continue
		}
		if len(principalSet) > 0 {
			if _, ok := principalSet[b.principalUserID]; !ok {
				continue
			}
		}
		if query.RoleName != "" && b.roleName != query.RoleName {
			continue
		}
		if query.Resource.Type != "" && b.resource.Type != query.Resource.Type {
			continue
		}
		if query.Resource.ID != "" && b.resource.ID != query.Resource.ID {
			continue
		}
		revoked := revokedAt
		b.revokedAt = &revoked
	}
	return nil
}

// ListRoleBindings mirrors GormGrantRepository's own matching rule: a wildcard-scoped binding
// (resource.ID == domain.WildcardResourceID) matches a concrete query.Resource.ID too.
func (r *memoryRoleBindingRepository) ListRoleBindings(_ context.Context, query domain.RoleBindingQuery) ([]domain.RoleBinding, error) {
	principalSet := make(map[string]struct{}, len(query.PrincipalUserIDs))
	for _, id := range query.PrincipalUserIDs {
		principalSet[id] = struct{}{}
	}
	var out []domain.RoleBinding
	for _, b := range r.bindings {
		if b.revokedAt != nil {
			continue
		}
		if len(principalSet) > 0 {
			if _, ok := principalSet[b.principalUserID]; !ok {
				continue
			}
		}
		if query.RoleName != "" && b.roleName != query.RoleName {
			continue
		}
		if query.Resource.Type != "" && b.resource.Type != query.Resource.Type {
			continue
		}
		if query.Resource.ID != "" && b.resource.ID != query.Resource.ID && b.resource.ID != domain.WildcardResourceID {
			continue
		}
		out = append(out, domain.RoleBinding{
			PrincipalUserID: b.principalUserID, RoleName: b.roleName, Resource: b.resource,
			GrantedByUserID: b.grantedByUserID, CreatedAt: b.createdAt,
		})
	}
	return out, nil
}

func newRoleBindingServiceFixture(t *testing.T) (*RoleBindingService, *memoryRoleBindingRepository, domain.Principal) {
	t.Helper()
	const now = int64(1_700_000_000)
	repo := &memoryRoleBindingRepository{}
	service := NewRoleBindingService(testRegistry(t), repo)
	service.now = func() int64 { return now }
	issuer := domain.Principal{
		Type: domain.PrincipalTypeUser, ID: "owner",
		GlobalPermissions: map[string]struct{}{testBoundary: {}},
	}
	return service, repo, issuer
}

func TestRoleBindingServiceAssignBatchesMultiplePrincipalsInOneCall(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	resource := domain.Resource{Type: testResource, ID: "doc1"}
	err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u2", "u1"}, RoleName: "EDITOR",
		Resource: resource, Context: domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}},
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if repo.assignCalls != 1 {
		t.Fatalf("assignCalls = %d, want exactly 1 batched call for 2 principals", repo.assignCalls)
	}
	got := append([]string(nil), repo.assignedBatch...)
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"u1", "u2"}) {
		t.Fatalf("assignedBatch = %#v, want both principals in one call", got)
	}
	if len(repo.bindings) != 2 {
		t.Fatalf("bindings created = %d, want 2", len(repo.bindings))
	}
	for _, b := range repo.bindings {
		if b.grantedByUserID != "owner" {
			t.Fatalf("binding grantedByUserID = %q, want issuer recorded", b.grantedByUserID)
		}
		if b.roleName != "EDITOR" || b.resource != resource {
			t.Fatalf("binding role/resource mismatch: %#v", b)
		}
	}
}

func TestRoleBindingServiceAssignSupportsWildcardResource(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "REVIEWER",
		Resource: domain.Resource{Type: testResource, ID: domain.WildcardResourceID},
		Context:  domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}},
	})
	if err != nil {
		t.Fatalf("Assign wildcard: %v", err)
	}
	if len(repo.bindings) != 1 || repo.bindings[0].resource.ID != domain.WildcardResourceID {
		t.Fatalf("wildcard binding not recorded: %#v", repo.bindings)
	}
}

func TestRoleBindingServiceAssignRejectsUnauthorizedIssuer(t *testing.T) {
	service, repo, _ := newRoleBindingServiceFixture(t)
	unauthorized := domain.Principal{
		Type: domain.PrincipalTypeUser, ID: "not-the-manager",
		GlobalPermissions: map[string]struct{}{testBoundary: {}},
	}
	err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: unauthorized, PrincipalUserIDs: []string{"u1"}, RoleName: "EDITOR",
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
		Context:  domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}},
	})
	if !errors.Is(err, domain.ErrGrantManagementForbidden) {
		t.Fatalf("Assign by unauthorized issuer error = %v, want ErrGrantManagementForbidden", err)
	}
	if len(repo.bindings) != 0 {
		t.Fatalf("no binding should have been created, got %#v", repo.bindings)
	}
}

func TestRoleBindingServiceAssignRejectsMissingGlobalBoundaryPermission(t *testing.T) {
	service, repo, _ := newRoleBindingServiceFixture(t)
	issuerWithoutBoundary := domain.Principal{Type: domain.PrincipalTypeUser, ID: "owner"}
	err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuerWithoutBoundary, PrincipalUserIDs: []string{"u1"}, RoleName: "EDITOR",
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
		Context:  domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}},
	})
	if !errors.Is(err, domain.ErrGrantManagementForbidden) {
		t.Fatalf("Assign without boundary permission error = %v, want ErrGrantManagementForbidden", err)
	}
	if len(repo.bindings) != 0 {
		t.Fatalf("no binding should have been created, got %#v", repo.bindings)
	}
}

func TestRoleBindingServiceRevokeEndsAccessForBatchedPrincipals(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	resource := domain.Resource{Type: testResource, ID: "doc1"}
	assignCtx := domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1", "u2"}, RoleName: "EDITOR",
		Resource: resource, Context: assignCtx,
	}); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := service.Revoke(context.Background(), domain.RoleBindingRevocation{
		Issuer: issuer, PrincipalUserIDs: []string{"u1", "u2"}, RoleName: "EDITOR",
		Resource: resource, Context: assignCtx,
	}); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	for _, b := range repo.bindings {
		if b.revokedAt == nil {
			t.Fatalf("binding %#v was not revoked", b)
		}
	}
}

// TestRoleBindingServiceRevokeAlreadyRevokedIsNoOp covers the role-binding-management spec's
// "Revoking an already-revoked binding is a no-op" scenario.
func TestRoleBindingServiceRevokeAlreadyRevokedIsNoOp(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	resource := domain.Resource{Type: testResource, ID: "doc1"}
	ctx := domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "EDITOR", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	revoke := domain.RoleBindingRevocation{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "EDITOR", Resource: resource, Context: ctx,
	}
	if err := service.Revoke(context.Background(), revoke); err != nil {
		t.Fatalf("first Revoke: %v", err)
	}
	firstRevokedAt := *repo.bindings[0].revokedAt
	if err := service.Revoke(context.Background(), revoke); err != nil {
		t.Fatalf("second Revoke (already revoked) should not error: %v", err)
	}
	if *repo.bindings[0].revokedAt != firstRevokedAt {
		t.Fatalf("revoking an already-revoked binding must leave its revocation state unchanged")
	}
}

// TestRoleBindingServiceRevokeResourceIgnoresPrincipalRoleAndAuthorization covers RevokeResource:
// it revokes every binding on a resource regardless of principal or role name, with no
// CanManageGrants check — mirroring GrantService.RevokeResource's own exemption, reserved for
// domain lifecycle cleanup (e.g. the resource itself was deleted).
func TestRoleBindingServiceRevokeResourceIgnoresPrincipalRoleAndAuthorization(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	resource := domain.Resource{Type: testResource, ID: "doc1"}
	ctx := domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1", "u2"}, RoleName: "EDITOR", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u3"}, RoleName: "REVIEWER", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign second role: %v", err)
	}
	if err := service.RevokeResource(context.Background(), resource); err != nil {
		t.Fatalf("RevokeResource: %v", err)
	}
	for _, b := range repo.bindings {
		if b.revokedAt == nil {
			t.Fatalf("binding %#v was not revoked by RevokeResource", b)
		}
	}
}

// TestRoleBindingServiceRevokeWithEmptyRoleNameEndsEveryRoleForPrincipal covers removing a
// principal's membership entirely (e.g. Course's RemoveCollaborator): an empty RoleName must not
// be rejected the way Assign rejects it, and must revoke every active role the principal holds
// on the resource, not just one.
func TestRoleBindingServiceRevokeWithEmptyRoleNameEndsEveryRoleForPrincipal(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	resource := domain.Resource{Type: testResource, ID: "doc1"}
	ctx := domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "EDITOR", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign EDITOR: %v", err)
	}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "REVIEWER", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign REVIEWER: %v", err)
	}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u2"}, RoleName: "EDITOR", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign EDITOR for a different principal: %v", err)
	}
	if err := service.Revoke(context.Background(), domain.RoleBindingRevocation{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Revoke with empty RoleName: %v", err)
	}
	for _, b := range repo.bindings {
		if b.principalUserID == "u1" && b.revokedAt == nil {
			t.Fatalf("u1's binding %#v should have been revoked regardless of role", b)
		}
		if b.principalUserID == "u2" && b.revokedAt != nil {
			t.Fatalf("u2's binding %#v must not be touched by a revoke scoped to u1", b)
		}
	}
}

func TestRoleBindingServiceAssignRejectsEmptyRoleName(t *testing.T) {
	service, repo, issuer := newRoleBindingServiceFixture(t)
	err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "",
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
	})
	if !errors.Is(err, domain.ErrInvalidRoleBinding) {
		t.Fatalf("Assign with empty RoleName error = %v, want ErrInvalidRoleBinding", err)
	}
	if len(repo.bindings) != 0 {
		t.Fatalf("no binding should have been created, got %#v", repo.bindings)
	}
}

// TestRoleBindingServiceListReturnsOnlyActiveBindingsMatchingQuery covers List's use for
// display (e.g. Course's collaborator list / instructor-candidate exclusion set): it must
// return active bindings matching the query's filters and omit revoked ones.
func TestRoleBindingServiceListReturnsOnlyActiveBindingsMatchingQuery(t *testing.T) {
	service, _, issuer := newRoleBindingServiceFixture(t)
	resource := domain.Resource{Type: testResource, ID: "doc1"}
	otherResource := domain.Resource{Type: testResource, ID: "doc2"}
	ctx := domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1", "u2"}, RoleName: "EDITOR", Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign u1/u2: %v", err)
	}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u3"}, RoleName: "EDITOR", Resource: otherResource, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign u3 on a different resource: %v", err)
	}
	if err := service.Revoke(context.Background(), domain.RoleBindingRevocation{
		Issuer: issuer, PrincipalUserIDs: []string{"u2"}, Resource: resource, Context: ctx,
	}); err != nil {
		t.Fatalf("Revoke u2: %v", err)
	}
	got, err := service.List(context.Background(), domain.RoleBindingQuery{Resource: resource})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].PrincipalUserID != "u1" {
		t.Fatalf("List(resource) = %#v, want only u1's active binding", got)
	}
}

// TestRoleBindingServiceListMatchesWildcardBindingForConcreteResource covers List's use by
// Course's collaborator listing / instructor-candidate exclusion set: a principal holding a
// wildcard EDITOR binding must still show up when listing for one specific resource, mirroring
// roleExpandedGrants' authorization-decision matching rule (a wildcard-bound principal already
// has effective access to that resource).
func TestRoleBindingServiceListMatchesWildcardBindingForConcreteResource(t *testing.T) {
	service, _, issuer := newRoleBindingServiceFixture(t)
	ctx := domain.EvaluationContext{DomainFacts: testPolicyFacts{ManagerID: "owner"}}
	if err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: []string{"u1"}, RoleName: "EDITOR",
		Resource: domain.Resource{Type: testResource, ID: domain.WildcardResourceID}, Context: ctx,
	}); err != nil {
		t.Fatalf("Assign wildcard: %v", err)
	}
	got, err := service.List(context.Background(), domain.RoleBindingQuery{
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].PrincipalUserID != "u1" {
		t.Fatalf("List(concrete resource) = %#v, want the wildcard-bound u1", got)
	}
}

func TestRoleBindingServiceAssignRejectsEmptyPrincipalList(t *testing.T) {
	service, _, issuer := newRoleBindingServiceFixture(t)
	err := service.Assign(context.Background(), domain.RoleBindingAssignment{
		Issuer: issuer, PrincipalUserIDs: nil, RoleName: "EDITOR",
		Resource: domain.Resource{Type: testResource, ID: "doc1"},
	})
	if !errors.Is(err, domain.ErrInvalidRoleBinding) {
		t.Fatalf("Assign with empty principal list error = %v, want ErrInvalidRoleBinding", err)
	}
}
