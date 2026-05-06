package media

import (
	"mycourse-io-be/pkg/errors"
)

func RequireInitialized[T any](dependency *T) error {
	if dependency == nil {
		return errors.ErrDependencyNotConfigured
	}
	return nil
}
