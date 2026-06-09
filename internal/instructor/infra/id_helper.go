package infra

import (
	"strings"

	"mycourse-io-be/internal/shared/uuidx"
)

func ensureStringID(id *string) error {
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
