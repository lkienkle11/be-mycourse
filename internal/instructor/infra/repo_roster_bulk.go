package infra

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
)

const rosterBulkInsertBatchSize = 100

type rosterUserRoleRow struct {
	UserID string `gorm:"type:uuid;primaryKey"`
	RoleID uint   `gorm:"primaryKey"`
}

func (rosterUserRoleRow) TableName() string { return constants.TableRBACUserRoles }

type rosterUserRow struct {
	ID           string `gorm:"column:id"`
	DisplayName  string `gorm:"column:display_name"`
	Email        string `gorm:"column:email"`
	Phone        string `gorm:"column:phone"`
	AvatarFileID string `gorm:"column:avatar_file_id"`
}

func (r *GormRepository) loadRosterUsersByIDs(
	ctx context.Context,
	db *gorm.DB,
	userIDs []string,
) (map[string]rosterUserRow, error) {
	rows, err := gormx.FindActiveByIDs[rosterUserRow](ctx, db, constants.TableAppUsers,
		`id, display_name, email, COALESCE(phone, '') AS phone,
COALESCE(avatar_file_id::text, '') AS avatar_file_id`, userIDs)
	if err != nil {
		return nil, err
	}
	return gormx.IndexByID(rows, func(row rosterUserRow) string { return row.ID }), nil
}

func platformStaffUserIDSet(ctx context.Context, db *gorm.DB, userIDs []string) (map[string]struct{}, error) {
	return gormx.UserIDSetByRoleNames(ctx, db, userIDs, []string{
		domain.RoleNameSysadmin,
		domain.RoleNameAdmin,
	})
}

func existingInstructorUserIDSet(
	ctx context.Context,
	db *gorm.DB,
	userIDs []string,
	instructorRoleID uint,
) (map[string]struct{}, error) {
	if len(userIDs) == 0 {
		return map[string]struct{}{}, nil
	}
	var matched []string
	err := db.WithContext(ctx).
		Table(constants.TableRBACUserRoles).
		Select("user_id").
		Where("user_id IN ? AND role_id = ?", userIDs, instructorRoleID).
		Scan(&matched).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(matched))
	for _, id := range matched {
		out[id] = struct{}{}
	}
	return out, nil
}

type bulkRosterWriteState struct {
	failed           []domain.RosterBulkFailure
	succeededUserIDs []string
	insertUserIDs    []string
}

func planBulkRosterWrites(
	userIDs []string,
	usersByID map[string]rosterUserRow,
	staffSet map[string]struct{},
	existingInstructorSet map[string]struct{},
) bulkRosterWriteState {
	state := bulkRosterWriteState{
		failed:           make([]domain.RosterBulkFailure, 0),
		succeededUserIDs: make([]string, 0, len(userIDs)),
		insertUserIDs:    make([]string, 0, len(userIDs)),
	}
	for _, userID := range userIDs {
		if _, ok := usersByID[userID]; !ok {
			state.failed = append(state.failed, domain.RosterBulkFailure{
				UserID: userID, Message: "user not found",
			})
			continue
		}
		if _, ok := staffSet[userID]; ok {
			state.failed = append(state.failed, domain.RosterBulkFailure{
				UserID: userID, Message: domain.ErrRosterPlatformStaffUser.Error(),
			})
			continue
		}
		state.succeededUserIDs = append(state.succeededUserIDs, userID)
		if _, ok := existingInstructorSet[userID]; !ok {
			state.insertUserIDs = append(state.insertUserIDs, userID)
		}
	}
	return state
}

func applyBulkRosterWrites(
	ctx context.Context,
	tx *gorm.DB,
	insertUserIDs []string,
	instructorRoleID uint,
) error {
	if len(insertUserIDs) == 0 {
		return nil
	}
	rows := make([]rosterUserRoleRow, 0, len(insertUserIDs))
	for _, userID := range insertUserIDs {
		rows = append(rows, rosterUserRoleRow{UserID: userID, RoleID: instructorRoleID})
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, rosterBulkInsertBatchSize).Error
}

func buildRosterMembersFromUsers(
	succeededUserIDs []string,
	usersByID map[string]rosterUserRow,
) []domain.RosterMember {
	out := make([]domain.RosterMember, 0, len(succeededUserIDs))
	for _, userID := range succeededUserIDs {
		u, ok := usersByID[userID]
		if !ok {
			continue
		}
		member := domain.RosterMember{
			UserID: u.ID, FullName: u.DisplayName, Email: u.Email, Phone: u.Phone,
		}
		if u.AvatarFileID != "" {
			member.AvatarFileID = u.AvatarFileID
		}
		out = append(out, member)
	}
	return out
}

func (r *GormRepository) AddRosterBulk(
	ctx context.Context,
	userIDs []string,
	instructorRoleID uint,
) (domain.RosterBulkResult, error) {
	result := domain.RosterBulkResult{
		Added:  make([]domain.RosterMember, 0, len(userIDs)),
		Failed: make([]domain.RosterBulkFailure, 0),
	}
	if len(userIDs) == 0 {
		return result, nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		usersByID, err := r.loadRosterUsersByIDs(ctx, tx, userIDs)
		if err != nil {
			return err
		}
		staffSet, err := platformStaffUserIDSet(ctx, tx, userIDs)
		if err != nil {
			return err
		}
		existingSet, err := existingInstructorUserIDSet(ctx, tx, userIDs, instructorRoleID)
		if err != nil {
			return err
		}
		writeState := planBulkRosterWrites(userIDs, usersByID, staffSet, existingSet)
		result.Failed = writeState.failed
		if len(writeState.insertUserIDs) > 0 {
			if err := applyBulkRosterWrites(ctx, tx, writeState.insertUserIDs, instructorRoleID); err != nil {
				return err
			}
		}
		if len(writeState.succeededUserIDs) == 0 {
			return nil
		}
		result.Added = buildRosterMembersFromUsers(writeState.succeededUserIDs, usersByID)
		result.InsertedUserIDs = append([]string(nil), writeState.insertUserIDs...)
		return nil
	})
	if err != nil {
		return domain.RosterBulkResult{}, err
	}
	return result, nil
}
