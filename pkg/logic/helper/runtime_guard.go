package helper

import (
	"errors"

	"mycourse-io-be/pkg/errcode"
)

var ErrDependencyNotConfigured = errors.New(errcode.DefaultMessage(errcode.InternalError))

func RequireInitialized[T any](dependency *T) error {
	if dependency == nil {
		return ErrDependencyNotConfigured
	}
	return nil
}
