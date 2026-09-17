package infra

import (
	"context"

	"gorm.io/gorm"

	authzdomain "mycourse-io-be/internal/authorization/domain"
	courseapp "mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	instructordomain "mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/useraccess"
)

func (r *GormRepository) instructorUserIDSet(ctx context.Context, db *gorm.DB, userIDs []string) (map[string]struct{}, error) {
	return gormx.UserIDSetByRoleNames(ctx, db, userIDs, []string{
		instructordomain.RoleNameInstructor,
		instructordomain.RoleNameSysadmin,
		instructordomain.RoleNameAdmin,
	})
}

// existingActiveCollaboratorUserIDs reports which of userIDs already hold an active role
// binding on this course, so planBulkCollaboratorWrites can tell "already a collaborator" (no
// write needed — bulk add only ever assigns EDITOR, and re-adding an existing collaborator
// can't change their role since EDITOR is the only role bulk add can request) from "brand new"
// (needs a fresh RoleBindingService.Assign). Reads through RoleBindingService.List, the one
// sanctioned read path for authorization_role_bindings — never a direct query against that
// table. tx joins the read into the caller's ongoing transaction via gormx.WithTx.
func (r *GormRepository) existingActiveCollaboratorUserIDs(ctx context.Context, tx *gorm.DB, courseID string, userIDs []string) (map[string]struct{}, error) {
	bindings, err := r.roleBindings.List(gormx.WithTx(ctx, tx), authzdomain.RoleBindingQuery{
		PrincipalUserIDs: userIDs,
		Resource:         authzdomain.Resource{Type: courseapp.ResourceTypeCourse, ID: courseID},
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(bindings))
	for _, b := range bindings {
		out[b.PrincipalUserID] = struct{}{}
	}
	return out, nil
}

type bulkCollaboratorWriteState struct {
	failed           []domain.CollaboratorBulkFailure
	succeededUserIDs []string
	insertUserIDs    []string
}

func planBulkCollaboratorWrites(
	userIDs []string,
	instructorSet map[string]struct{},
	accessByID map[string]useraccess.AssignmentSnapshot,
	existingUserIDs map[string]struct{},
	now int64,
) bulkCollaboratorWriteState {
	state := bulkCollaboratorWriteState{
		failed:           make([]domain.CollaboratorBulkFailure, 0),
		succeededUserIDs: make([]string, 0, len(userIDs)),
		insertUserIDs:    make([]string, 0, len(userIDs)),
	}
	for _, userID := range userIDs {
		if _, ok := instructorSet[userID]; !ok {
			state.failed = append(state.failed, domain.CollaboratorBulkFailure{
				UserID:  userID,
				Message: domain.ErrCourseInstructorRequired.Error(),
			})
			continue
		}
		snap, ok := accessByID[userID]
		if !ok {
			state.failed = append(state.failed, domain.CollaboratorBulkFailure{
				UserID:  userID,
				Message: useraccess.ErrUserNotFound.Error(),
			})
			continue
		}
		if err := useraccess.CheckAccessible(&snap.Snapshot, now); err != nil {
			state.failed = append(state.failed, domain.CollaboratorBulkFailure{
				UserID:  userID,
				Message: domain.ErrCourseCollaboratorInactive.Error(),
			})
			continue
		}
		if _, ok := existingUserIDs[userID]; !ok {
			state.insertUserIDs = append(state.insertUserIDs, userID)
		}
		state.succeededUserIDs = append(state.succeededUserIDs, userID)
	}
	return state
}

// AddCollaboratorsBulk validates and plans inside one course-side transaction (access check,
// instructor eligibility, existing-binding lookup — a consistent read snapshot), then assigns
// the role binding for every newly-added principal in one batched RoleBindingService.Assign
// call (never one call per principal, per .ai/skills/logic-n-1-optimize) as part of that same
// transaction: gormx.WithTx makes internal/authorization's GormGrantRepository (which owns its
// own *gorm.DB, constructed once at wiring time) join tx instead of running a separately
// committed write, so a failure after the plan step rolls the whole operation back atomically.
func (r *GormRepository) AddCollaboratorsBulk(
	ctx context.Context,
	courseID string,
	actorUserID string,
	userIDs []string,
	role string,
) (domain.CollaboratorBulkResult, error) {
	result := domain.CollaboratorBulkResult{
		Added:  make([]domain.Collaborator, 0, len(userIDs)),
		Failed: make([]domain.CollaboratorBulkFailure, 0),
	}
	if len(userIDs) == 0 {
		return result, nil
	}
	var plan bulkCollaboratorWriteState
	var resolvedCourseID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID, courseapp.ActionCourseCollaboratorsManage)
		if err != nil {
			return err
		}
		resolvedCourseID = access.ID
		instructorSet, err := r.instructorUserIDSet(ctx, tx, userIDs)
		if err != nil {
			return err
		}
		existingUserIDs, err := r.existingActiveCollaboratorUserIDs(ctx, tx, access.ID, userIDs)
		if err != nil {
			return err
		}
		accessByID, err := gormx.LoadAssignmentSnapshotsByIDs(ctx, tx, userIDs)
		if err != nil {
			return err
		}
		plan = planBulkCollaboratorWrites(userIDs, instructorSet, accessByID, existingUserIDs, timex.NowUnix())
		if len(plan.insertUserIDs) == 0 {
			return nil
		}
		return r.roleBindings.Assign(gormx.WithTx(ctx, tx), authzdomain.RoleBindingAssignment{
			Issuer:           courseAuthorizationPrincipal(ctx, actorUserID),
			PrincipalUserIDs: plan.insertUserIDs,
			RoleName:         role,
			Resource:         authzdomain.Resource{Type: courseapp.ResourceTypeCourse, ID: resolvedCourseID},
			Context:          authzdomain.EvaluationContext{DomainFacts: courseapp.CourseAuthorizationFacts{IsOwner: true}},
		})
	})
	if err != nil {
		return domain.CollaboratorBulkResult{}, err
	}
	result.Failed = plan.failed
	if len(plan.succeededUserIDs) == 0 {
		return result, nil
	}
	added, err := r.loadCollaboratorsByUserIDs(ctx, r.db, resolvedCourseID, plan.succeededUserIDs)
	if err != nil {
		return domain.CollaboratorBulkResult{}, err
	}
	result.Added = added
	return result, nil
}
