package media

import (
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/query"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) fileListBaseQuery(filter dto.FileFilter) *gorm.DB {
	q := r.db.Model(&models.MediaFile{}).Where("deleted_at IS NULL")
	if filter.Provider != nil && *filter.Provider != "" {
		q = q.Where("provider = ?", *filter.Provider)
	}
	if filter.Kind != nil && *filter.Kind != "" {
		q = q.Where("kind = ?", *filter.Kind)
	}
	if where, arg, ok := query.BuildSearchClause(filter.BaseFilter, map[string]string{
		"filename":   "filename",
		"object_key": "object_key",
		"mime_type":  "mime_type",
		"status":     "status",
	}); ok {
		q = q.Where(where, arg)
	}
	return q
}

func (r *FileRepository) List(filter dto.FileFilter) ([]models.MediaFile, int64, error) {
	q := r.fileListBaseQuery(filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	p := query.ParseListFilter(filter.BaseFilter)
	sortClause := query.BuildSortClause(filter.BaseFilter, map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"filename":   "filename",
		"size_bytes": "size_bytes",
	}, "created_at")
	var rows []models.MediaFile
	if err := q.Order(sortClause).Offset(p.Offset).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *FileRepository) GetByObjectKey(objectKey string) (*models.MediaFile, error) {
	var row models.MediaFile
	if err := r.db.Where("object_key = ? AND deleted_at IS NULL", objectKey).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FileRepository) GetByID(id string) (*models.MediaFile, error) {
	var row models.MediaFile
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FileRepository) GetByBunnyVideoID(videoID string) (*models.MediaFile, error) {
	var row models.MediaFile
	if err := r.db.Where("bunny_video_id = ? AND deleted_at IS NULL", videoID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FileRepository) UpsertByObjectKey(row *models.MediaFile) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.MediaFile
		err := tx.Where("object_key = ?", row.ObjectKey).First(&existing).Error
		if err == nil {
			row.ID = existing.ID
			row.CreatedAt = existing.CreatedAt
			row.RowVersion = existing.RowVersion + 1
			return tx.Model(&existing).Select("*").Updates(row).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if row.RowVersion <= 0 {
			row.RowVersion = 1
		}
		return tx.Create(row).Error
	})
}

func (r *FileRepository) SaveWithRowVersionCheck(row *models.MediaFile, expectedVersion int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var cur models.MediaFile
		if err := tx.Where("id = ?", row.ID).First(&cur).Error; err != nil {
			return err
		}
		if cur.RowVersion != expectedVersion {
			return pkgerrors.ErrMediaOptimisticLock
		}
		row.RowVersion = expectedVersion + 1
		row.CreatedAt = cur.CreatedAt
		return tx.Model(&cur).Select("*").Updates(row).Error
	})
}

// GetByURL returns the first non-deleted media_files row whose public URL or
// origin URL matches the given value. Used by orphan-cleanup flows to resolve
// stored provider/key information from a plain image URL field on another entity.
func (r *FileRepository) GetByURL(url string) (*models.MediaFile, error) {
	var row models.MediaFile
	if err := r.db.
		Where("(url = ? OR origin_url = ?) AND deleted_at IS NULL", url, url).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FileRepository) SoftDeleteByObjectKey(objectKey string) error {
	return r.db.Model(&models.MediaFile{}).
		Where("object_key = ? AND deleted_at IS NULL", objectKey).
		Update("deleted_at", gorm.Expr("NOW()")).
		Error
}
