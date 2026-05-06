package auth

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	authcache "mycourse-io-be/services/cache"
	mediasvc "mycourse-io-be/services/media"
)

func loadUserForAvatarUpdate(userID uint) (models.User, error) {
	var user models.User
	if err := models.DB.Preload("AvatarFile").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, pkgerrors.ErrUserNotFound
		}
		return models.User{}, err
	}
	return user, nil
}

func resolveAvatarFileIDForUpdate(in *string) (dbValue *string, compareNext string, err error) {
	next := strings.TrimSpace(*in)
	if next == "" {
		return nil, "", nil
	}
	if _, err := mediasvc.LoadValidatedProfileImageFile(next); err != nil {
		return nil, "", err
	}
	return &next, next, nil
}

func maybeEnqueueReplacedAvatar(prev, next string) {
	if prev != "" && prev != next {
		mediasvc.EnqueueOrphanCleanupForMediaFileID(prev)
	}
}

// UpdateMe applies PATCH /api/v1/me fields (currently avatar_file_id only).
func UpdateMe(userID uint, req dto.UpdateMeRequest) (*dto.MeResponse, error) {
	ctx := context.Background()
	if req.AvatarFileID == nil {
		return GetMe(userID)
	}

	user, err := loadUserForAvatarUpdate(userID)
	if err != nil {
		return nil, err
	}

	var prev string
	if user.AvatarFileID != nil {
		prev = strings.TrimSpace(*user.AvatarFileID)
	}

	nextPtr, nextCanon, err := resolveAvatarFileIDForUpdate(req.AvatarFileID)
	if err != nil {
		return nil, err
	}

	if err := models.DB.Model(&models.User{}).Where("id = ?", userID).
		Update("avatar_file_id", nextPtr).Error; err != nil {
		return nil, err
	}

	maybeEnqueueReplacedAvatar(prev, nextCanon)
	authcache.DelCachedUserMe(ctx, userID)

	return GetMe(userID)
}
