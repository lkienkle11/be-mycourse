package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	instructordomain "mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/useraccess"
)

const collaboratorBulkInsertBatchSize = 100

func (r *GormRepository) instructorUserIDSet(ctx context.Context, db *gorm.DB, userIDs []string) (map[string]struct{}, error) {
	return gormx.UserIDSetByRoleNames(ctx, db, userIDs, []string{
		instructordomain.RoleNameInstructor,
		instructordomain.RoleNameSysadmin,
		instructordomain.RoleNameAdmin,
	})
}

type bulkCollaboratorWriteState struct {
	failed           []domain.CollaboratorBulkFailure
	succeededUserIDs []string
	updateIDs        []string
	insertUserIDs    []string
}

func planBulkCollaboratorWrites(
	userIDs []string,
	instructorSet map[string]struct{},
	accessByID map[string]useraccess.AssignmentSnapshot,
	existingByUser map[string]collaboratorRow,
	now int64,
) bulkCollaboratorWriteState {
	state := bulkCollaboratorWriteState{
		failed:           make([]domain.CollaboratorBulkFailure, 0),
		succeededUserIDs: make([]string, 0, len(userIDs)),
		updateIDs:        make([]string, 0),
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
		if existing, ok := existingByUser[userID]; ok {
			state.updateIDs = append(state.updateIDs, existing.ID)
		} else {
			state.insertUserIDs = append(state.insertUserIDs, userID)
		}
		state.succeededUserIDs = append(state.succeededUserIDs, userID)
	}
	return state
}

func applyBulkCollaboratorWrites(
	ctx context.Context,
	tx *gorm.DB,
	courseID string,
	userIDs []string,
	role string,
	instructorSet map[string]struct{},
	accessByID map[string]useraccess.AssignmentSnapshot,
	existingByUser map[string]collaboratorRow,
) (bulkCollaboratorWriteState, error) {
	now := timex.NowUnix()
	plan := planBulkCollaboratorWrites(userIDs, instructorSet, accessByID, existingByUser, now)
	if len(plan.updateIDs) == 0 && len(plan.insertUserIDs) == 0 {
		return plan, nil
	}
	if len(plan.updateIDs) > 0 {
		if err := tx.WithContext(ctx).Model(&collaboratorRow{}).
			Where("id IN ? AND deleted_at IS NULL", plan.updateIDs).
			Updates(map[string]any{"role": role, "updated_at": now}).Error; err != nil {
			return bulkCollaboratorWriteState{}, err
		}
	}
	if len(plan.insertUserIDs) > 0 {
		rows := make([]collaboratorRow, 0, len(plan.insertUserIDs))
		for _, userID := range plan.insertUserIDs {
			row := collaboratorRow{CourseID: courseID, UserID: userID, Role: role}
			if err := ensureCourseRowID(&row); err != nil {
				return bulkCollaboratorWriteState{}, err
			}
			gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
			rows = append(rows, row)
		}
		if err := tx.WithContext(ctx).CreateInBatches(rows, collaboratorBulkInsertBatchSize).Error; err != nil {
			return bulkCollaboratorWriteState{}, err
		}
	}
	return plan, nil
}

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
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		instructorSet, err := r.instructorUserIDSet(ctx, tx, userIDs)
		if err != nil {
			return err
		}
		var existingRows []collaboratorRow
		if err := tx.Where("course_id = ? AND user_id IN ? AND deleted_at IS NULL", access.ID, userIDs).Find(&existingRows).Error; err != nil {
			return err
		}
		existingByUser := make(map[string]collaboratorRow, len(existingRows))
		for _, row := range existingRows {
			existingByUser[row.UserID] = row
		}
		accessByID, err := gormx.LoadAssignmentSnapshotsByIDs(ctx, tx, userIDs)
		if err != nil {
			return err
		}
		writeState, err := applyBulkCollaboratorWrites(ctx, tx, access.ID, userIDs, role, instructorSet, accessByID, existingByUser)
		if err != nil {
			return err
		}
		result.Failed = writeState.failed
		if len(writeState.succeededUserIDs) == 0 {
			return nil
		}
		added, err := r.loadCollaboratorsByUserIDs(ctx, tx, access.ID, writeState.succeededUserIDs)
		if err != nil {
			return err
		}
		result.Added = added
		return nil
	})
	if err != nil {
		return domain.CollaboratorBulkResult{}, err
	}
	return result, nil
}
