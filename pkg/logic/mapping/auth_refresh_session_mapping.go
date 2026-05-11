package mapping

import (
	"mycourse-io-be/pkg/entities"
	gormjsonbauth "mycourse-io-be/pkg/gormjsonb/auth"
)

// ToRefreshTokenSessionEntity converts the GORM JSONB carrier to the domain session map (Rule 14 — gormjsonb ↔ entities in mapping only).
func ToRefreshTokenSessionEntity(m gormjsonbauth.RefreshTokenSessionMap) entities.RefreshTokenSessionMap {
	out := make(entities.RefreshTokenSessionMap)
	for k, v := range m {
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
