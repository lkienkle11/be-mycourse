package infra

import (
	"errors"
	"reflect"
	"testing"

	"mycourse-io-be/internal/authorization/domain"
)

func TestValidatePersistedActionRowsRejectsResourceMismatch(t *testing.T) {
	actions := []domain.ActionDefinition{{Name: "document:update", ResourceType: "document"}}
	if err := validatePersistedActionRows(actions, []actionRow{{ActionName: "document:update", ResourceType: "document"}}); err != nil {
		t.Fatalf("matching catalog: %v", err)
	}
	err := validatePersistedActionRows(actions, []actionRow{{ActionName: "document:update", ResourceType: "course"}})
	if !errors.Is(err, domain.ErrActionCatalogConflict) {
		t.Fatalf("mismatch error = %v", err)
	}
}

func TestReplacementRowsDeduplicatePrincipalActionPairs(t *testing.T) {
	rows, err := replacementRows(domain.GrantReplacement{
		PrincipalUserIDs: []string{"u2", "u1", "u1"},
		Actions:          []string{"document:read", "document:update", "document:update"},
		Resource:         domain.Resource{Type: "document", ID: "doc-1"},
		Effect:           domain.EffectAllow,
	}, 1_700_000_000)
	if err != nil {
		t.Fatalf("replacementRows: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want four unique principal/action pairs", len(rows))
	}
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := row.PrincipalUserID + ":" + row.ActionName
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate pair %s", key)
		}
		seen[key] = struct{}{}
	}
}

func TestMergeGrantsEmpty(t *testing.T) {
	if got := mergeGrants(nil, nil); len(got) != 0 {
		t.Fatalf("mergeGrants(nil, nil) = %#v, want empty", got)
	}
}

func TestMergeGrantsSingleSource(t *testing.T) {
	grant := domain.Grant{ID: "g1", PrincipalUserID: "u1", ActionName: "a1", Effect: domain.EffectAllow}
	roleGrant := domain.Grant{ID: "role:b1:a1", PrincipalUserID: "u1", ActionName: "a1", Effect: domain.EffectAllow}
	cases := []struct {
		name         string
		direct       []domain.Grant
		roleExpanded []domain.Grant
		wantID       string
	}{
		{name: "direct only", direct: []domain.Grant{grant}, wantID: "g1"},
		{name: "role-expanded only", roleExpanded: []domain.Grant{roleGrant}, wantID: "role:b1:a1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeGrants(tc.direct, tc.roleExpanded)
			if len(got) != 1 || got[0].ID != tc.wantID {
				t.Fatalf("mergeGrants(%#v, %#v) = %#v, want single entry %q", tc.direct, tc.roleExpanded, got, tc.wantID)
			}
		})
	}
}

func TestMergeGrantsOrdersDeterministically(t *testing.T) {
	direct := []domain.Grant{
		{ID: "g-z", PrincipalUserID: "u2", ActionName: "a1", Effect: domain.EffectAllow},
		{ID: "g-a", PrincipalUserID: "u1", ActionName: "a2", Effect: domain.EffectAllow},
	}
	roleExpanded := []domain.Grant{
		{ID: "role:b1:a1", PrincipalUserID: "u1", ActionName: "a1", Effect: domain.EffectAllow},
	}
	got := mergeGrants(direct, roleExpanded)
	want := []string{"role:b1:a1", "g-a", "g-z"}
	if len(got) != len(want) {
		t.Fatalf("mergeGrants length = %d, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("mergeGrants[%d].ID = %q, want %q (full: %#v)", i, got[i].ID, id, got)
		}
	}
}

func TestMergeGrantsKeepsBothWhenDirectAndRoleExpandedOverlap(t *testing.T) {
	direct := []domain.Grant{{ID: "g1", PrincipalUserID: "u1", ActionName: "a1", Effect: domain.EffectAllow}}
	roleExpanded := []domain.Grant{{ID: "role:b1:a1", PrincipalUserID: "u1", ActionName: "a1", Effect: domain.EffectAllow}}
	got := mergeGrants(direct, roleExpanded)
	if len(got) != 2 {
		t.Fatalf("mergeGrants overlap = %#v, want both entries kept (no dedup)", got)
	}
}

func TestRoleExpandedRowToGrant(t *testing.T) {
	grantedBy := "granter-1"
	row := roleExpandedGrantRow{
		BindingID: "binding-1", PrincipalUserID: "u1", ResourceType: "course", ResourceID: "course-1",
		ActionName: "course_basic_info:update", GrantedByUserID: &grantedBy, CreatedAt: 1_700_000_000,
	}
	got := roleExpandedRowToGrant(&row)
	want := domain.Grant{
		ID: "role:binding-1:course_basic_info:update", PrincipalUserID: "u1",
		ActionName: "course_basic_info:update", ResourceType: "course", ResourceID: "course-1",
		Effect: domain.EffectAllow, GrantedByUserID: "granter-1", CreatedAt: 1_700_000_000,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roleExpandedRowToGrant = %#v, want %#v", got, want)
	}
	if got.ValidFrom != nil || got.ExpiresAt != nil || len(got.Conditions) != 0 || got.RevokedAt != nil {
		t.Fatalf("roleExpandedRowToGrant should leave ValidFrom/ExpiresAt/Conditions/RevokedAt at zero value, got %#v", got)
	}
}

// TestRoleExpandedRowToGrantSupportsMultipleIndependentRoles exercises the spec requirement
// "A resource type may define multiple, independently-membered role names": two role-name
// grants for the same principal/resource, each resolving to a different action, must remain
// distinguishable after mapping and merging (no collapsing into one entry).
func TestRoleExpandedRowToGrantSupportsMultipleIndependentRoles(t *testing.T) {
	editorRow := roleExpandedGrantRow{
		BindingID: "binding-editor", PrincipalUserID: "u1", ResourceType: "course", ResourceID: "course-1",
		ActionName: "course_basic_info:update", CreatedAt: 1_700_000_000,
	}
	reviewerRow := roleExpandedGrantRow{
		BindingID: "binding-reviewer", PrincipalUserID: "u1", ResourceType: "course", ResourceID: "course-1",
		ActionName: "course_review_note:create", CreatedAt: 1_700_000_000,
	}
	roleExpanded := []domain.Grant{roleExpandedRowToGrant(&editorRow), roleExpandedRowToGrant(&reviewerRow)}
	got := mergeGrants(nil, roleExpanded)
	if len(got) != 2 {
		t.Fatalf("mergeGrants two independent role grants = %#v, want both kept distinct", got)
	}
	byAction := make(map[string]domain.Grant, len(got))
	for _, g := range got {
		byAction[g.ActionName] = g
	}
	if g, ok := byAction["course_basic_info:update"]; !ok || g.ID != "role:binding-editor:course_basic_info:update" {
		t.Fatalf("missing/incorrect editor-role grant: %#v", byAction)
	}
	if g, ok := byAction["course_review_note:create"]; !ok || g.ID != "role:binding-reviewer:course_review_note:create" {
		t.Fatalf("missing/incorrect reviewer-role grant: %#v", byAction)
	}
}

func TestGrantConditionJSONRoundTrip(t *testing.T) {
	want := []domain.Condition{
		{Operator: domain.ConditionStringEquals, Key: "tenant", Values: []any{"alpha", "beta"}},
		{Operator: domain.ConditionBoolEquals, Key: "enabled", Values: []any{true}},
	}
	raw, err := marshalConditions(want)
	if err != nil {
		t.Fatalf("marshalConditions: %v", err)
	}
	got, err := unmarshalConditions(raw)
	if err != nil {
		t.Fatalf("unmarshalConditions: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("conditions = %#v, want %#v", got, want)
	}
}
