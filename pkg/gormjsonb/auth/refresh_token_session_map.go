package auth

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
)

// RefreshTokenSessionMap is the Postgres JSONB column type for users.refresh_token_session.
type RefreshTokenSessionMap entities.RefreshTokenSessionMap

func (m RefreshTokenSessionMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]entities.RefreshSessionEntry(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *RefreshTokenSessionMap) Scan(src any) error {
	if src == nil {
		*m = RefreshTokenSessionMap{}
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
		*m = RefreshTokenSessionMap{}
		return nil
	}
	out := make(entities.RefreshTokenSessionMap)
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = RefreshTokenSessionMap(out)
	return nil
}
