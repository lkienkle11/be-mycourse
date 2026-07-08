package infra

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/constants"
)

// metadataJSONB is the Postgres JSONB carrier for user_oauth_identities.metadata.
type metadataJSONB map[string]any

func (m metadataJSONB) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *metadataJSONB) Scan(src any) error {
	if src == nil {
		*m = metadataJSONB{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported metadata JSONB source type %T", src)
	}
	if len(b) == 0 || string(b) == "null" {
		*m = metadataJSONB{}
		return nil
	}
	out := make(map[string]any)
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = metadataJSONB(out)
	return nil
}

// oauthIdentityRow is the GORM model for the user_oauth_identities table.
type oauthIdentityRow struct {
	ID            string        `gorm:"type:uuid;primaryKey"`
	UserID        string        `gorm:"column:user_id;type:uuid;not null;index"`
	Provider      string        `gorm:"size:32;not null"`
	ProviderSub   string        `gorm:"column:provider_sub;size:255;not null"`
	ProviderEmail *string       `gorm:"column:provider_email;size:255"`
	LinkedAt      int64         `gorm:"column:linked_at;not null"`
	LastLoginAt   *int64        `gorm:"column:last_login_at"`
	Metadata      metadataJSONB `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt     int64         `gorm:"column:created_at;not null"`
	UpdatedAt     int64         `gorm:"column:updated_at;not null"`
}

func (oauthIdentityRow) TableName() string { return constants.TableUserOAuthIdentities }

func toOAuthIdentityDomain(r *oauthIdentityRow) *domain.UserOAuthIdentity {
	return &domain.UserOAuthIdentity{
		ID:            r.ID,
		UserID:        r.UserID,
		Provider:      domain.OAuthProvider(r.Provider),
		ProviderSub:   r.ProviderSub,
		ProviderEmail: r.ProviderEmail,
		LinkedAt:      r.LinkedAt,
		LastLoginAt:   r.LastLoginAt,
		Metadata:      map[string]any(r.Metadata),
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func toOAuthIdentityRow(i *domain.UserOAuthIdentity) *oauthIdentityRow {
	return &oauthIdentityRow{
		ID:            i.ID,
		UserID:        i.UserID,
		Provider:      string(i.Provider),
		ProviderSub:   i.ProviderSub,
		ProviderEmail: i.ProviderEmail,
		LinkedAt:      i.LinkedAt,
		LastLoginAt:   i.LastLoginAt,
		Metadata:      metadataJSONB(i.Metadata),
		CreatedAt:     i.CreatedAt,
		UpdatedAt:     i.UpdatedAt,
	}
}
