// Package infra contains MEDIA bounded-context infrastructure: GORM repos, cloud clients.
package infra

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
)

// mediaFileRow is the GORM model for the media_files table.
type mediaFileRow struct {
	ID                 string     `gorm:"column:id;type:uuid;primaryKey"`
	ObjectKey          string     `gorm:"column:object_key;type:varchar(512);uniqueIndex;not null"`
	Kind               string     `gorm:"column:kind;type:varchar(16);not null"`
	Provider           string     `gorm:"column:provider;type:varchar(16);not null"`
	Filename           string     `gorm:"column:filename;type:varchar(512);not null"`
	MimeType           string     `gorm:"column:mime_type;type:varchar(255);not null;default:''"`
	SizeBytes          int64      `gorm:"column:size_bytes;not null;default:0"`
	URL                string     `gorm:"column:url;type:text;not null"`
	OriginURL          string     `gorm:"column:origin_url;type:text;not null"`
	Status             string     `gorm:"column:status;type:varchar(16);not null"`
	B2BucketName       string     `gorm:"column:b2_bucket_name;type:varchar(255);not null;default:''"`
	BunnyVideoID       string     `gorm:"column:bunny_video_id;type:varchar(255)"`
	BunnyLibraryID     string     `gorm:"column:bunny_library_id;type:varchar(255)"`
	VideoID            string     `gorm:"column:video_id;type:varchar(255);not null;default:''"`
	ThumbnailURL       string     `gorm:"column:thumbnail_url;type:text;not null;default:''"`
	EmbededHTML        string     `gorm:"column:embeded_html;type:text;not null;default:''"`
	Duration           int64      `gorm:"column:duration;not null;default:0"`
	VideoProvider      string     `gorm:"column:video_provider;type:varchar(64);not null;default:''"`
	RowVersion         int64      `gorm:"column:row_version;not null;default:1"`
	ContentFingerprint string     `gorm:"column:content_fingerprint;type:varchar(128);not null;default:''"`
	MetadataJSON       []byte     `gorm:"column:metadata_json;type:jsonb;not null;default:'{}'::jsonb"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt          *time.Time `gorm:"column:deleted_at"`
}

func (mediaFileRow) TableName() string { return constants.TableMediaFiles }

// pendingCleanupRow is the GORM model for media_pending_cloud_cleanup.
type pendingCleanupRow struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Provider     string    `gorm:"column:provider;type:varchar(16);not null"`
	ObjectKey    string    `gorm:"column:object_key;type:varchar(512);not null;default:''"`
	BunnyVideoID string    `gorm:"column:bunny_video_id;type:varchar(255);not null;default:''"`
	Status       string    `gorm:"column:status;type:varchar(32);not null"`
	AttemptCount int       `gorm:"column:attempt_count;not null;default:0"`
	LastError    string    `gorm:"column:last_error;type:text;not null;default:''"`
	NextRunAt    time.Time `gorm:"column:next_run_at;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (pendingCleanupRow) TableName() string { return constants.TableMediaPendingCloudCleanup }

// --- mappers -----------------------------------------------------------------

func rowToFile(r *mediaFileRow) *domain.File {
	f := &domain.File{
		ID: r.ID, ObjectKey: r.ObjectKey, Kind: r.Kind, Provider: r.Provider,
		Filename: r.Filename, MimeType: r.MimeType, SizeBytes: r.SizeBytes,
		URL: r.URL, OriginURL: r.OriginURL, Status: r.Status,
		B2BucketName: r.B2BucketName, BunnyVideoID: r.BunnyVideoID, BunnyLibraryID: r.BunnyLibraryID,
		VideoID: r.VideoID, ThumbnailURL: r.ThumbnailURL, EmbededHTML: r.EmbededHTML,
		Duration: r.Duration, VideoProvider: r.VideoProvider, RowVersion: r.RowVersion,
		ContentFingerprint: r.ContentFingerprint, MetadataJSON: string(r.MetadataJSON),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	// Re-derive the typed Metadata struct from the stored JSONB blob so that
	// reads (List / GetByID / GetByObjectKey / GetByBunnyVideoID) return the
	// same shape as the write path. Without this rebuild, the API response
	// always shows zero values for width/height/duration/codec/... even when
	// `metadata_json` holds rich Bunny telemetry.
	raw := f.RawMetadataMap()
	if raw == nil {
		raw = domain.RawMetadata{}
	}
	// Payload is nil on read-back; buildImageTypedMetadata is resilient and
	// falls back to width/height stored inside the raw map.
	f.Metadata = BuildTypedMetadata(r.Kind, r.MimeType, r.Filename, r.SizeBytes, nil, raw)
	return f
}

func fileToRow(f *domain.File) *mediaFileRow {
	metaJSON := f.MetadataJSONBytes()
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}
	return &mediaFileRow{
		ID: f.ID, ObjectKey: f.ObjectKey, Kind: f.Kind, Provider: f.Provider,
		Filename: f.Filename, MimeType: f.MimeType, SizeBytes: f.SizeBytes,
		URL: f.URL, OriginURL: f.OriginURL, Status: f.Status,
		B2BucketName: f.B2BucketName, BunnyVideoID: f.BunnyVideoID, BunnyLibraryID: f.BunnyLibraryID,
		VideoID: f.VideoID, ThumbnailURL: f.ThumbnailURL, EmbededHTML: f.EmbededHTML,
		Duration: f.Duration, VideoProvider: f.VideoProvider, RowVersion: f.RowVersion,
		ContentFingerprint: f.ContentFingerprint, MetadataJSON: metaJSON,
		CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}

// --- GormFileRepository ------------------------------------------------------

// GormFileRepository implements domain.FileRepository.
type GormFileRepository struct{ db *gorm.DB }

func NewGormFileRepository(db *gorm.DB) *GormFileRepository { return &GormFileRepository{db: db} }

func (r *GormFileRepository) List(ctx context.Context, filter domain.FileFilter) ([]domain.File, int64, error) {
	q := r.db.WithContext(ctx).Model(&mediaFileRow{}).Where("deleted_at IS NULL")
	if filter.Provider != nil {
		q = q.Where("provider = ?", *filter.Provider)
	}
	if filter.Kind != nil {
		q = q.Where("kind = ?", *filter.Kind)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	var rows []mediaFileRow
	if err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.File, len(rows))
	for i := range rows {
		out[i] = *rowToFile(&rows[i])
	}
	return out, total, nil
}

func (r *GormFileRepository) GetByID(ctx context.Context, id string) (*domain.File, error) {
	var row mediaFileRow
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return rowToFile(&row), nil
}

func (r *GormFileRepository) GetByObjectKey(ctx context.Context, objectKey string) (*domain.File, error) {
	var row mediaFileRow
	if err := r.db.WithContext(ctx).Where("object_key = ? AND deleted_at IS NULL", objectKey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return rowToFile(&row), nil
}

func (r *GormFileRepository) GetByBunnyVideoID(ctx context.Context, videoGUID string) (*domain.File, error) {
	var row mediaFileRow
	if err := r.db.WithContext(ctx).Where("bunny_video_id = ? AND deleted_at IS NULL", videoGUID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return rowToFile(&row), nil
}

func (r *GormFileRepository) UpsertByObjectKey(ctx context.Context, f *domain.File) error {
	row := fileToRow(f)
	return r.db.WithContext(ctx).
		Where("object_key = ?", row.ObjectKey).
		Assign(row).
		FirstOrCreate(row).Error
}

func (r *GormFileRepository) SaveWithRowVersionCheck(ctx context.Context, f *domain.File, expectedVersion int64) error {
	row := fileToRow(f)
	result := r.db.WithContext(ctx).
		Model(&mediaFileRow{}).
		Where("id = ? AND row_version = ? AND deleted_at IS NULL", f.ID, expectedVersion).
		Updates(map[string]any{
			"object_key": row.ObjectKey, "kind": row.Kind, "provider": row.Provider,
			"filename": row.Filename, "mime_type": row.MimeType, "size_bytes": row.SizeBytes,
			"url": row.URL, "origin_url": row.OriginURL, "status": row.Status,
			"b2_bucket_name": row.B2BucketName, "bunny_video_id": row.BunnyVideoID,
			"bunny_library_id": row.BunnyLibraryID, "video_id": row.VideoID,
			"thumbnail_url": row.ThumbnailURL, "embeded_html": row.EmbededHTML,
			"duration": row.Duration, "video_provider": row.VideoProvider,
			"content_fingerprint": row.ContentFingerprint, "metadata_json": row.MetadataJSON,
			"updated_at": time.Now(),
			"row_version": gorm.Expr("row_version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrMediaOptimisticLock
	}
	return nil
}

func (r *GormFileRepository) SoftDeleteByObjectKey(ctx context.Context, objectKey string) error {
	return r.db.WithContext(ctx).
		Model(&mediaFileRow{}).
		Where("object_key = ? AND deleted_at IS NULL", objectKey).
		Update("deleted_at", time.Now()).Error
}

// --- GormPendingCleanupRepository --------------------------------------------

// GormPendingCleanupRepository implements domain.PendingCleanupRepository.
type GormPendingCleanupRepository struct{ db *gorm.DB }

func NewGormPendingCleanupRepository(db *gorm.DB) *GormPendingCleanupRepository {
	return &GormPendingCleanupRepository{db: db}
}

func (r *GormPendingCleanupRepository) Create(ctx context.Context, rec *domain.MediaPendingCloudCleanup) error {
	row := &pendingCleanupRow{
		Provider: rec.Provider, ObjectKey: rec.ObjectKey, BunnyVideoID: rec.BunnyVideoID,
		Status: "PENDING", NextRunAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormPendingCleanupRepository) FindPending(ctx context.Context, limit int) ([]*domain.MediaPendingCloudCleanup, error) {
	var rows []pendingCleanupRow
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_run_at <= ?", "PENDING", time.Now()).
		Order("next_run_at ASC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.MediaPendingCloudCleanup, len(rows))
	for i, row := range rows {
		out[i] = &domain.MediaPendingCloudCleanup{
			ID: row.ID, Provider: row.Provider, ObjectKey: row.ObjectKey,
			BunnyVideoID: row.BunnyVideoID, Status: row.Status,
			AttemptCount: row.AttemptCount, LastError: row.LastError,
			NextRunAt: row.NextRunAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

func (r *GormPendingCleanupRepository) MarkDone(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&pendingCleanupRow{}, id).Error
}

func (r *GormPendingCleanupRepository) MarkFailed(ctx context.Context, id int64, errMsg string, _ interface{}) error {
	return r.db.WithContext(ctx).Model(&pendingCleanupRow{}).Where("id = ?", id).Updates(map[string]any{
		"status": "FAILED", "last_error": errMsg, "updated_at": time.Now(),
	}).Error
}

func (r *GormPendingCleanupRepository) DeleteByObjectKey(ctx context.Context, objectKey string) error {
	return r.db.WithContext(ctx).Where("object_key = ?", objectKey).Delete(&pendingCleanupRow{}).Error
}

