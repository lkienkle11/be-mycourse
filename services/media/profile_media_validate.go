package media

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/mapping"
)

// LoadValidatedProfileImageFile resolves fileID to a non-deleted media_files row suitable
// for taxonomy cover images or user avatars (FILE kind, READY, raster image MIME / extension).
func LoadValidatedProfileImageFile(fileID string) (*models.MediaFile, error) {
	id := strings.TrimSpace(fileID)
	if id == "" {
		return nil, pkgerrors.ErrInvalidProfileMediaFile
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, pkgerrors.ErrInvalidProfileMediaFile
	}
	row, err := mediaRepository().GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.ErrInvalidProfileMediaFile
		}
		return nil, err
	}
	if row.Status != constants.FileStatusReady {
		return nil, pkgerrors.ErrInvalidProfileMediaFile
	}
	if !mapping.ProfileImageFileAcceptable(string(row.Kind), row.MimeType, row.Filename) {
		return nil, pkgerrors.ErrInvalidProfileMediaFile
	}
	return row, nil
}
