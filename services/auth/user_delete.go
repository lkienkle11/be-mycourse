package auth

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	jobmedia "mycourse-io-be/internal/jobs/media"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	authcache "mycourse-io-be/services/cache"
)

// SoftDeleteUserWithAvatarCleanup soft-deletes the user (GORM DeletedAt) and schedules
// deferred cloud cleanup for avatar_file_id when present. Call from admin flows or tests.
func SoftDeleteUserWithAvatarCleanup(userID uint) error {
	ctx := context.Background()
	var user models.User
	if err := models.DB.Preload("AvatarFile").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.ErrUserNotFound
		}
		return err
	}
	var avatarFID string
	if user.AvatarFileID != nil {
		avatarFID = strings.TrimSpace(*user.AvatarFileID)
	}
	if err := models.DB.Delete(&user).Error; err != nil {
		return err
	}
	authcache.DelCachedUserMe(ctx, userID)
	if avatarFID != "" {
		jobmedia.EnqueueOrphanCleanupForMediaFileID(avatarFID)
	}
	return nil
}
