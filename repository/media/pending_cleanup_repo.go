package media

import (
	"time"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
)

func (r *FileRepository) InsertPendingCleanup(row *models.MediaPendingCloudCleanup) error {
	if row == nil {
		return nil
	}
	row.Status = constants.PendingCleanupStatusPending
	row.NextRunAt = time.Now()
	return r.db.Create(row).Error
}

func (r *FileRepository) ListPendingCleanupDue(limit int) ([]models.MediaPendingCloudCleanup, error) {
	var rows []models.MediaPendingCloudCleanup
	err := r.db.
		Where("status = ? AND next_run_at <= ?", constants.PendingCleanupStatusPending, time.Now()).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *FileRepository) MarkPendingCleanupDone(id int64) error {
	return r.db.Delete(&models.MediaPendingCloudCleanup{}, id).Error
}

func (r *FileRepository) MarkPendingCleanupFailed(id int64, msg string) error {
	return r.db.Model(&models.MediaPendingCloudCleanup{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     constants.PendingCleanupStatusFailed,
			"last_error": msg,
			"updated_at": time.Now(),
		}).Error
}

func (r *FileRepository) SchedulePendingCleanupRetry(id int64, msg string, attempt int, next time.Time) error {
	return r.db.Model(&models.MediaPendingCloudCleanup{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempt_count": attempt,
			"last_error":    msg,
			"next_run_at":   next,
			"updated_at":    time.Now(),
		}).Error
}
