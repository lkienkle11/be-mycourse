package domain

import "context"

// UserRepository defines data access operations for the User aggregate.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByUserCode(ctx context.Context, userCode string) (*User, error)
	FindByConfirmationToken(ctx context.Context, token string) (*User, error)
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
	UpdateDisplayName(ctx context.Context, userID string, displayName string) error
	UpdateAvatar(ctx context.Context, userID string, avatarFileID *string) error
	SoftDelete(ctx context.Context, userID string) error
	HardDelete(ctx context.Context, userID string) error
}

// RefreshSessionRepository manages the refresh_token_session JSONB column on users.
type RefreshSessionRepository interface {
	// LoadSessions returns the session map for a user.
	LoadSessions(ctx context.Context, userID string) (RefreshTokenSessionMap, error)
	// SaveSessions overwrites the session map atomically.
	SaveSessions(ctx context.Context, userID string, sessions RefreshTokenSessionMap) error
	// RemoveSession deletes one session key from the user's refresh_token_session JSONB.
	RemoveSession(ctx context.Context, userID string, sessionStr string) error
	// IncrementEmailSendCount increments the registration_email_send_total counter and returns the new value.
	IncrementEmailSendCount(ctx context.Context, userID string) (int, error)
}
