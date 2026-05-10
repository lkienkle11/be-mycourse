package auth

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
)

// sessionColumnJSONB is the local JSONB wrapper type (Rule 11: methods only on a type defined in this package).
type sessionColumnJSONB entities.RefreshTokenSessionMap

// RefreshTokenSessionMap is the Postgres JSONB column type for users.refresh_token_session (exported alias).
type RefreshTokenSessionMap = sessionColumnJSONB

func (m sessionColumnJSONB) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]entities.RefreshSessionEntry(m))
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
	out := make(entities.RefreshTokenSessionMap)
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = sessionColumnJSONB(out)
	return nil
}
