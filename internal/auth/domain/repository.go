package domain

import "context"

// UserRepository defines data access operations for the User aggregate.
type UserRepository interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUserCode(ctx context.Context, userCode string) (*User, error)
	FindByConfirmationToken(ctx context.Context, token string) (*User, error)
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
	UpdateDisplayName(ctx context.Context, userID uint, displayName string) error
	UpdateAvatar(ctx context.Context, userID uint, avatarFileID *string) error
	SoftDelete(ctx context.Context, userID uint) error
}

// RefreshSessionRepository manages the refresh_token_session JSONB column on users.
type RefreshSessionRepository interface {
	// LoadSessions returns the session map for a user.
	LoadSessions(ctx context.Context, userID uint) (RefreshTokenSessionMap, error)
	// SaveSessions overwrites the session map atomically.
	SaveSessions(ctx context.Context, userID uint, sessions RefreshTokenSessionMap) error
	// IncrementEmailSendCount increments the registration_email_send_total counter and returns the new value.
	IncrementEmailSendCount(ctx context.Context, userID uint) (int, error)
}
