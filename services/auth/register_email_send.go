package auth

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/brevo"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/setting"
	authcache "mycourse-io-be/services/cache"
)

func enforceRegistrationEmailLifetimeOrAbandon(ctx context.Context, norm string, user *models.User) error {
	if user.RegistrationEmailSendTotal < constants.MaxRegisterConfirmationEmailsLifetime {
		return nil
	}
	if err := models.DB.Unscoped().Delete(&models.User{}, user.ID).Error; err != nil {
		return err
	}
	authcache.DeleteRegisterConfirmationEmailWindow(ctx, user.ID)
	authcache.DelCachedLoginUserByNormalizedEmail(ctx, norm)
	return pkgerrors.ErrRegistrationAbandoned
}

func persistRegistrationEmailSendSuccess(userID uint) error {
	return models.DB.Model(&models.User{}).Where("id = ?", userID).
		Update("registration_email_send_total", gorm.Expr("registration_email_send_total + ?", 1)).Error
}

func deliverRegistrationConfirmationEmail(
	ctx context.Context,
	norm, email, displayName string,
	user *models.User,
	reservationID string,
) error {
	if user.ConfirmationToken == nil {
		authcache.ReleaseRegisterConfirmationSend(ctx, user.ID, reservationID)
		return pkgerrors.ErrConfirmationEmailSendFailed
	}
	confirmURL := setting.AppSetting.AppBaseURL + "/api/v1/auth/confirm?token=" + *user.ConfirmationToken
	if err := brevo.SendConfirmationEmail(email, displayName, confirmURL); err != nil {
		authcache.ReleaseRegisterConfirmationSend(ctx, user.ID, reservationID)
		return pkgerrors.ErrConfirmationEmailSendFailed
	}
	if err := persistRegistrationEmailSendSuccess(user.ID); err != nil {
		return err
	}
	return nil
}
