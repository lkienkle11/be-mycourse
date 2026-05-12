package domain

import "context"

// FileFilter defines query parameters for listing media files.
type FileFilter struct {
	Page     int
	PageSize int
	Provider *string
	Kind     *string
}

// FileRepository defines persistence operations for the File aggregate.
type FileRepository interface {
	List(ctx context.Context, filter FileFilter) ([]File, int64, error)
	GetByID(ctx context.Context, id string) (*File, error)
	GetByObjectKey(ctx context.Context, objectKey string) (*File, error)
	GetByBunnyVideoID(ctx context.Context, videoGUID string) (*File, error)
	UpsertByObjectKey(ctx context.Context, f *File) error
	SaveWithRowVersionCheck(ctx context.Context, f *File, expectedVersion int64) error
	SoftDeleteByObjectKey(ctx context.Context, objectKey string) error
}

// PendingCleanupRepository manages deferred cloud-storage cleanup records.
type PendingCleanupRepository interface {
	Create(ctx context.Context, r *MediaPendingCloudCleanup) error
	FindPending(ctx context.Context, limit int) ([]*MediaPendingCloudCleanup, error)
	MarkDone(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, errMsg string, nextRunAt interface{}) error
	DeleteByObjectKey(ctx context.Context, objectKey string) error
}
