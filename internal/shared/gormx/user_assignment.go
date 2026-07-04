package gormx

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/useraccess"
)

type assignmentSnapshotRow struct {
	ID             string `gorm:"column:id"`
	DeletedAt      *int64 `gorm:"column:deleted_at"`
	IsDisable      bool   `gorm:"column:is_disable"`
	BannedUntil    *int64 `gorm:"column:banned_until"`
	EmailConfirmed bool   `gorm:"column:email_confirmed"`
}

func rowToAssignmentSnapshot(row assignmentSnapshotRow) useraccess.AssignmentSnapshot {
	return useraccess.AssignmentSnapshot{
		Snapshot: useraccess.Snapshot{
			DeletedAt:   row.DeletedAt,
			IsDisabled:  row.IsDisable,
			BannedUntil: row.BannedUntil,
		},
		EmailConfirmed: row.EmailConfirmed,
	}
}

// LoadAssignmentSnapshotsByIDs batch-loads assignment eligibility snapshots for user IDs.
func LoadAssignmentSnapshotsByIDs(
	ctx context.Context,
	db *gorm.DB,
	userIDs []string,
) (map[string]useraccess.AssignmentSnapshot, error) {
	if len(userIDs) == 0 {
		return map[string]useraccess.AssignmentSnapshot{}, nil
	}
	var rows []assignmentSnapshotRow
	if err := db.WithContext(ctx).
		Table(constants.TableAppUsers).
		Select("id, deleted_at, is_disable, banned_until, email_confirmed").
		Where("id IN ?", userIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]useraccess.AssignmentSnapshot, len(rows))
	for _, row := range rows {
		out[row.ID] = rowToAssignmentSnapshot(row)
	}
	return out, nil
}

// LoadAssignmentSnapshotByID loads one assignment eligibility snapshot.
func LoadAssignmentSnapshotByID(
	ctx context.Context,
	db *gorm.DB,
	userID string,
) (useraccess.AssignmentSnapshot, error) {
	var row assignmentSnapshotRow
	err := db.WithContext(ctx).
		Table(constants.TableAppUsers).
		Select("id, deleted_at, is_disable, banned_until, email_confirmed").
		Where("id = ?", userID).
		Take(&row).Error
	if err != nil {
		return useraccess.AssignmentSnapshot{}, err
	}
	return rowToAssignmentSnapshot(row), nil
}
