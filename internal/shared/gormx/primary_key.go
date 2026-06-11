package gormx

import (
	"strings"

	"mycourse-io-be/internal/shared/uuidx"
)

// EnsureStringID assigns a UUID v7 when id is empty. Used before GORM Create on
// string primary keys so zero values are not inserted as "".
func EnsureStringID(id *string) error {
	if strings.TrimSpace(*id) != "" {
		return nil
	}
	v7, err := uuidx.NewV7()
	if err != nil {
		return err
	}
	*id = v7
	return nil
}
