package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	authcache "mycourse-io-be/services/cache"
)

// Register validates input, creates a new pending user or updates an existing unconfirmed user,
// then sends a confirmation email subject to lifetime and Redis window limits.
func Register(email, password, displayName string) error {
	if !isStrongPassword(password) {
		return pkgerrors.ErrWeakPassword
	}
	ctx := context.Background()
	norm := authcache.NormalizeLoginEmail(email)

	var existing models.User
	err := models.DB.Where("email = ?", email).First(&existing).Error
	if err == nil {
		if existing.EmailConfirmed {
			return pkgerrors.ErrEmailAlreadyExists
		}
		return registerResendPending(ctx, norm, email, password, displayName, &existing)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return registerNewPending(ctx, norm, email, password, displayName)
}

func registerNewPending(ctx context.Context, norm, email, password, displayName string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	uc, err := uuid.NewV7()
	if err != nil {
		return err
	}
	confirmToken := uuid.New().String()
	now := time.Now()
	user := models.User{
		UserCode:                   uc.String(),
		Email:                      email,
		HashPassword:               string(hash),
		DisplayName:                displayName,
		RegistrationEmailSendTotal: 0,
		ConfirmationToken:          &confirmToken,
		ConfirmationSentAt:         &now,
	}
	if err := models.DB.Create(&user).Error; err != nil {
		return err
	}
	return sendRegistrationConfirmationEmail(ctx, norm, email, displayName, &user)
}

func registerResendPending(ctx context.Context, norm, email, password, displayName string, existing *models.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	newTok := uuid.New().String()
	now := time.Now()
	updates := map[string]interface{}{
		"hash_password":        string(hash),
		"display_name":         displayName,
		"confirmation_token":   newTok,
		"confirmation_sent_at": now,
	}
	if err := models.DB.Model(existing).Updates(updates).Error; err != nil {
		return err
	}
	var user models.User
	if err := models.DB.First(&user, existing.ID).Error; err != nil {
		return err
	}
	return sendRegistrationConfirmationEmail(ctx, norm, email, displayName, &user)
}

func sendRegistrationConfirmationEmail(ctx context.Context, norm, email, displayName string, user *models.User) error {
	if err := enforceRegistrationEmailLifetimeOrAbandon(ctx, norm, user); err != nil {
		return err
	}

	allowed, retryAfter, reservationID, err := authcache.TryReserveRegisterConfirmationSend(ctx, user.ID)
	if err != nil {
		return err
	}
	if !allowed {
		sec := int64(retryAfter / time.Second)
		if sec < 1 {
			sec = 1
		}
		return &pkgerrors.RegistrationEmailRateLimitedError{RetryAfterSeconds: sec}
	}

	return deliverRegistrationConfirmationEmail(ctx, norm, email, displayName, user, reservationID)
}
