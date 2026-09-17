package application

import (
	"errors"
	"testing"

	authzdomain "mycourse-io-be/internal/authorization/domain"
)

// TestCoursePolicyProviderEvaluate covers both the owner-allow path (synthesized PolicyAllow,
// no stored role binding needed) and the non-owner path (PolicyNeutral, passthrough to
// role-expanded grants), as one table so the two near-identical cases aren't flagged as
// duplicate test bodies.
func TestCoursePolicyProviderEvaluate(t *testing.T) {
	cases := []struct {
		name       string
		isOwner    bool
		wantEffect authzdomain.PolicyEffect
	}{
		{name: "owner is allowed outright", isOwner: true, wantEffect: authzdomain.PolicyAllow},
		{name: "non-owner is neutral, passthrough to role-expanded grants", isOwner: false, wantEffect: authzdomain.PolicyNeutral},
	}
	provider := NewCoursePolicyProvider()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := provider.Evaluate(authzdomain.AuthorizationRequest{
				Context: authzdomain.EvaluationContext{DomainFacts: CourseAuthorizationFacts{IsOwner: tc.isOwner}},
			})
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if decision.Effect != tc.wantEffect {
				t.Fatalf("decision effect = %q, want %q", decision.Effect, tc.wantEffect)
			}
		})
	}
}

func TestCoursePolicyProviderEvaluateRejectsMissingFacts(t *testing.T) {
	provider := NewCoursePolicyProvider()
	_, err := provider.Evaluate(authzdomain.AuthorizationRequest{})
	if !errors.Is(err, authzdomain.ErrInvalidPolicyContext) {
		t.Fatalf("Evaluate without CourseAuthorizationFacts error = %v, want ErrInvalidPolicyContext", err)
	}
}

func TestCoursePolicyProviderCanManageGrantsOwnerOnly(t *testing.T) {
	provider := NewCoursePolicyProvider()
	owner, err := provider.CanManageGrants(authzdomain.GrantManagementRequest{
		Context: authzdomain.EvaluationContext{DomainFacts: CourseAuthorizationFacts{IsOwner: true}},
	})
	if err != nil || !owner {
		t.Fatalf("CanManageGrants(owner) = %v, %v, want true, nil", owner, err)
	}
	nonOwner, err := provider.CanManageGrants(authzdomain.GrantManagementRequest{
		Context: authzdomain.EvaluationContext{DomainFacts: CourseAuthorizationFacts{IsOwner: false}},
	})
	if err != nil || nonOwner {
		t.Fatalf("CanManageGrants(non-owner) = %v, %v, want false, nil", nonOwner, err)
	}
}

func TestCoursePolicyProviderResourceTypeAndActionsAreConsistent(t *testing.T) {
	provider := NewCoursePolicyProvider()
	if provider.ResourceType() != ResourceTypeCourse {
		t.Fatalf("ResourceType() = %q, want %q", provider.ResourceType(), ResourceTypeCourse)
	}
	actions := provider.Actions()
	if len(actions) == 0 {
		t.Fatal("Actions() must not be empty")
	}
	seen := make(map[string]bool, len(actions))
	for _, action := range actions {
		if action.ResourceType != ResourceTypeCourse {
			t.Fatalf("action %q has ResourceType %q, want %q", action.Name, action.ResourceType, ResourceTypeCourse)
		}
		if action.BoundaryPermission == "" {
			t.Fatalf("action %q has empty BoundaryPermission", action.Name)
		}
		seen[action.Name] = true
	}
	// Every action RoleActions() references must be declared in Actions(), so the
	// authorization_role_actions FK into authorization_actions is always satisfiable.
	for _, roleAction := range RoleActions() {
		if !seen[roleAction.ActionName] {
			t.Fatalf("RoleActions() references undeclared action %q", roleAction.ActionName)
		}
	}
}

func TestRoleActionsMatchesTodaysOwnerEditorBehaviorExactly(t *testing.T) {
	byRole := map[string]map[string]bool{}
	for _, ra := range RoleActions() {
		if byRole[ra.RoleName] == nil {
			byRole[ra.RoleName] = map[string]bool{}
		}
		byRole[ra.RoleName][ra.ActionName] = true
	}
	anyCollaborator := []string{
		ActionCourseDetailView, ActionCourseDraftEdit, ActionCourseLeaseManage,
		ActionCourseCollaboratorsView, ActionCourseReviewView,
	}
	ownerOnly := []string{
		ActionCourseDraftPrepare, ActionCourseLifecycleDelete,
		ActionCourseCollaboratorsManage, ActionCourseReviewManage,
	}
	for _, action := range anyCollaborator {
		if !byRole["OWNER"][action] {
			t.Fatalf("OWNER must cover any-collaborator action %q (today: any active collaborator passes)", action)
		}
		if !byRole["EDITOR"][action] {
			t.Fatalf("EDITOR must cover any-collaborator action %q (today: any active collaborator passes)", action)
		}
	}
	for _, action := range ownerOnly {
		if !byRole["OWNER"][action] {
			t.Fatalf("OWNER must cover owner-only action %q", action)
		}
		if byRole["EDITOR"][action] {
			t.Fatalf("EDITOR must NOT cover owner-only action %q (today: only the owner passes)", action)
		}
	}
}
