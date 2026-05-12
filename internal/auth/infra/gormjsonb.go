// Package infra contains AUTH bounded-context infrastructure: GORM repos, Supabase client, crypto.
package infra

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/constants"
)

// sessionColumnJSONB is the local JSONB wrapper type for the refresh_token_session column.
type sessionColumnJSONB domain.RefreshTokenSessionMap

// RefreshTokenSessionMap is the Postgres JSONB column type wrapping domain.RefreshTokenSessionMap.
type RefreshTokenSessionMap sessionColumnJSONB

func (m sessionColumnJSONB) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]domain.RefreshSessionEntry(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *sessionColumnJSONB) Scan(src any) error {
	if src == nil {
		*m = sessionColumnJSONB{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf(constants.MsgRefreshSessionUnsupportedType, src)
	}
	if len(b) == 0 || string(b) == "null" {
		*m = sessionColumnJSONB{}
		return nil
	}
	out := make(domain.RefreshTokenSessionMap)
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = sessionColumnJSONB(out)
	return nil
}

func (m RefreshTokenSessionMap) Value() (driver.Value, error) {
	return sessionColumnJSONB(m).Value()
}

func (m *RefreshTokenSessionMap) Scan(src any) error {
	var u sessionColumnJSONB
	if err := (&u).Scan(src); err != nil {
		return err
	}
	*m = RefreshTokenSessionMap(u)
	return nil
}

// toDomainSessionMap converts the GORM JSONB carrier to the domain session map.
func toDomainSessionMap(m RefreshTokenSessionMap) domain.RefreshTokenSessionMap {
	out := make(domain.RefreshTokenSessionMap)
	for k, v := range m {
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
