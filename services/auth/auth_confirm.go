package auth

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	authcache "mycourse-io-be/services/cache"
)

func loadUserByConfirmationToken(confirmToken string) (models.User, error) {
	var user models.User
	if dbErr := models.DB.Where("confirmation_token = ?", confirmToken).First(&user).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return models.User{}, pkgerrors.ErrInvalidConfirmToken
		}
		return models.User{}, dbErr
	}
	return user, nil
}

func runConfirmEmailTx(user *models.User) error {
	updates := map[string]interface{}{
		"email_confirmed":               true,
		"confirmation_token":            nil,
		"registration_email_send_total": 0,
	}
	return models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Updates(updates).Error; err != nil {
			return err
		}
		var learner models.Role
		if err := tx.Where("name = ?", constants.Role.Learner).First(&learner).Error; err != nil {
			return err
		}
		ur := models.UserRole{UserID: user.ID, RoleID: learner.ID}
		return tx.FirstOrCreate(&ur, models.UserRole{UserID: user.ID, RoleID: learner.ID}).Error
	})
}

func clearCachesAfterEmailConfirmed(ctx context.Context, user models.User) {
	norm := authcache.NormalizeLoginEmail(user.Email)
	authcache.DeleteRegisterConfirmationEmailWindow(ctx, user.ID)
	authcache.DelCachedLoginUserByNormalizedEmail(ctx, norm)
}
