package entities

// MeProfile is the service-layer /me projection: same JSON field names as dto.MeResponse for Redis cache.
// Avatar uses entities.MediaFilePublic (dto.MediaFilePublic aliases it — no duplicate struct).
type MeProfile struct {
	UserID         uint             `json:"user_id"`
	UserCode       string           `json:"user_code"`
	Email          string           `json:"email"`
	DisplayName    string           `json:"display_name"`
	Avatar         *MediaFilePublic `json:"avatar,omitempty"`
	EmailConfirmed bool             `json:"email_confirmed"`
	IsDisabled     bool             `json:"is_disabled"`
	CreatedAt      int64            `json:"created_at"`
	Permissions    []string         `json:"permissions"`
}
