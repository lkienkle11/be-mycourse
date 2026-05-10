package media

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
	pkgmedia "mycourse-io-be/pkg/media"
	mediarepo "mycourse-io-be/repository/media"
)

// DeleteFilesByObjectKeys deletes up to MaxMediaBatchDelete media rows by object_key (all-or-nothing validation:
// every key must exist). Deletes cloud objects then soft-deletes DB rows.
func DeleteFilesByObjectKeys(keys []string) error {
	if len(keys) == 0 {
		return pkgerrors.ErrBatchDeleteEmptyKeys
	}
	if len(keys) > constants.MaxMediaBatchDelete {
		return pkgerrors.ErrMediaBatchDeleteTooManyIDs
	}
	uniq, err := dedupeBatchDeleteKeys(keys)
	if err != nil {
		return err
	}
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return err
	}
	clients := pkgmedia.Cloud
	repo := mediaRepository()

	rows, err := loadMediaRowsForKeys(repo, uniq)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if err := deleteOneStoredMediaRow(clients, repo, row); err != nil {
			return err
		}
	}
	return nil
}

func dedupeBatchDeleteKeys(keys []string) ([]string, error) {
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			return nil, pkgerrors.ErrMediaObjectKeyRequired
		}
		if _, ok := seen[k]; ok {
			return nil, pkgerrors.ErrMediaDuplicateObjectKeysInBatchDelete
		}
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out, nil
}

func loadMediaRowsForKeys(repo *mediarepo.FileRepository, uniq []string) ([]*models.MediaFile, error) {
	rows := make([]*models.MediaFile, 0, len(uniq))
	for _, key := range uniq {
		row, err := repo.GetByObjectKey(key)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, pkgerrors.ErrMediaFileNotFoundForObjectKey
			}
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func deleteOneStoredMediaRow(clients *entities.CloudClients, repo *mediarepo.FileRepository, row *models.MediaFile) error {
	key := strings.TrimSpace(row.ObjectKey)
	provider := strings.TrimSpace(row.Provider)
	bunnyID := strings.TrimSpace(row.BunnyVideoID)
	if provider == "" {
		provider = pkgmedia.DefaultMediaProvider(row.Kind)
	}
	if err := pkgmedia.DeleteStoredObject(context.Background(), clients, key, provider, bunnyID); err != nil {
		return err
	}
	return repo.SoftDeleteByObjectKey(key)
}
