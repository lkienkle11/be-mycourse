package infra

import (
	"mycourse-io-be/internal/shared/errors"
)

func RequireInitialized[T any](dependency *T) error {
	if dependency == nil {
		return errors.ErrDependencyNotConfigured
	}
	return nil
}
