package application

import (
	authzdomain "mycourse-io-be/internal/authorization/domain"
	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/constants"
)

// ResourceTypeCourse is the authorization_actions/authorization_role_bindings resource_type
// value for every Course-scoped action.
const ResourceTypeCourse = "course"

// Course actions, one per access-check call site in internal/course/infra/repo_access.go and
// friends (see openspec/changes/replace-course-collaborator-with-role-gate/tasks.md task 3.1's
// enumeration). Each is either "any active collaborator" (OWNER and EDITOR both cover it in
// RoleActions below) or "owner-only" (OWNER only), matching today's behavior exactly.
const (
	ActionCourseDetailView          = "course_detail:view"          // any collaborator: loadCourseDetail
	ActionCourseDraftEdit           = "course_draft:edit"           // any collaborator: ensureEditableDraft
	ActionCourseLeaseManage         = "course_lease:manage"         // any collaborator: AcquireLease
	ActionCourseCollaboratorsView   = "course_collaborators:view"   // any collaborator: ListCollaborators
	ActionCourseReviewView          = "course_review:view"          // any collaborator: ListReviewHistory
	ActionCourseDraftPrepare        = "course_draft:prepare"        // owner-only: PrepareDraft
	ActionCourseLifecycleDelete     = "course_lifecycle:delete"     // owner-only: DeleteCourse
	ActionCourseCollaboratorsManage = "course_collaborators:manage" // owner-only: AddCollaboratorsBulk, RemoveCollaborator, ListInstructorCandidates
	ActionCourseReviewManage        = "course_review:manage"        // owner-only: updateDraftStatus (SubmitForReview et al.), ReopenDraft
)

// CourseAuthorizationFacts is the fact CoursePolicyProvider.Evaluate/CanManageGrants requires
// via AuthorizationRequest.Context.DomainFacts / GrantManagementRequest.Context.DomainFacts.
// Whether the principal is the course's registered owner requires a DB lookup
// (courses.owner_user_id) that happens at the call site (internal/course/infra), before
// Authorize/CanManageGrants is invoked — the provider itself never queries the database.
type CourseAuthorizationFacts struct {
	IsOwner bool
}

// CoursePolicyProvider registers Course's actions with internal/authorization and answers
// whether the course's registered owner should be allowed an action outright (synthesized,
// with no stored role-binding row) and who may manage Course role bindings/grants (owner only).
// It never compares a role name: the owner-allow decision is not "does this principal hold role
// OWNER" but "is this principal courses.owner_user_id", a fact resolved by the caller and passed
// in via CourseAuthorizationFacts.
type CoursePolicyProvider struct{}

func NewCoursePolicyProvider() *CoursePolicyProvider { return &CoursePolicyProvider{} }

func (CoursePolicyProvider) ResourceType() string { return ResourceTypeCourse }

func (CoursePolicyProvider) Actions() []authzdomain.ActionDefinition {
	return []authzdomain.ActionDefinition{
		{Name: ActionCourseDetailView, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseRead, Grantable: true, Description: "View full course detail"},
		{Name: ActionCourseDraftEdit, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseUpdate, Grantable: true, Description: "Edit the course draft (basic info, outline)"},
		{Name: ActionCourseLeaseManage, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseUpdate, Grantable: true, Description: "Acquire an outline edit lease"},
		{Name: ActionCourseCollaboratorsView, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseRead, Grantable: true, Description: "List course collaborators"},
		{Name: ActionCourseReviewView, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseRead, Grantable: true, Description: "View course review history"},
		{Name: ActionCourseDraftPrepare, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseUpdate, Grantable: true, Description: "Prepare a new draft version"},
		{Name: ActionCourseLifecycleDelete, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseDelete, Grantable: true, Description: "Delete or trash the course"},
		{Name: ActionCourseCollaboratorsManage, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseUpdate, Grantable: true, Description: "Add/remove course collaborators"},
		{Name: ActionCourseReviewManage, ResourceType: ResourceTypeCourse, BoundaryPermission: constants.AllPermissions.CourseUpdate, Grantable: true, Description: "Submit/reopen the course draft for review"},
	}
}

func (CoursePolicyProvider) Evaluate(request authzdomain.AuthorizationRequest) (authzdomain.PolicyDecision, error) {
	facts, ok := request.Context.DomainFacts.(CourseAuthorizationFacts)
	if !ok {
		return authzdomain.PolicyDecision{}, authzdomain.ErrInvalidPolicyContext
	}
	if facts.IsOwner {
		return authzdomain.PolicyDecision{Effect: authzdomain.PolicyAllow, Reason: authzdomain.ReasonProviderAllow}, nil
	}
	return authzdomain.PolicyDecision{Effect: authzdomain.PolicyNeutral}, nil
}

func (CoursePolicyProvider) CanManageGrants(request authzdomain.GrantManagementRequest) (bool, error) {
	facts, ok := request.Context.DomainFacts.(CourseAuthorizationFacts)
	if !ok {
		return false, authzdomain.ErrInvalidPolicyContext
	}
	return facts.IsOwner, nil
}

// RoleActions declares which of Actions() each course collaborator role currently covers, for
// startup seeding of authorization_role_actions via GrantRepository.SyncRoleActions.
// internal/authorization has no role-name concept of its own; this mapping is Course's own
// declarative data, the same way Actions() declares Course's action catalog. It preserves
// today's exact behavior: EDITOR covers everything an "any active collaborator" check allows
// today, OWNER covers that plus everything "owner-only" today.
func RoleActions() []authzdomain.RoleActionDefinition {
	anyCollaborator := []string{
		ActionCourseDetailView, ActionCourseDraftEdit, ActionCourseLeaseManage,
		ActionCourseCollaboratorsView, ActionCourseReviewView,
	}
	ownerOnly := []string{
		ActionCourseDraftPrepare, ActionCourseLifecycleDelete,
		ActionCourseCollaboratorsManage, ActionCourseReviewManage,
	}
	out := make([]authzdomain.RoleActionDefinition, 0, len(anyCollaborator)*2+len(ownerOnly))
	for _, action := range anyCollaborator {
		out = append(out,
			authzdomain.RoleActionDefinition{RoleName: domain.CollaboratorRoleOwner, ResourceType: ResourceTypeCourse, ActionName: action},
			authzdomain.RoleActionDefinition{RoleName: domain.CollaboratorRoleEditor, ResourceType: ResourceTypeCourse, ActionName: action},
		)
	}
	for _, action := range ownerOnly {
		out = append(out, authzdomain.RoleActionDefinition{RoleName: domain.CollaboratorRoleOwner, ResourceType: ResourceTypeCourse, ActionName: action})
	}
	return out
}
