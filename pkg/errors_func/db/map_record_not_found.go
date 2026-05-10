// Package errfuncdb holds database-adjacent error helper functions (Rule 19).
package errfuncdb

import (
	stderrors "errors"

	"gorm.io/gorm"

	pkgerrors "mycourse-io-be/pkg/errors"
)

// MapRecordNotFound translates gorm.ErrRecordNotFound into ErrNotFound.
func MapRecordNotFound(err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerrors.ErrNotFound
	}
	return err
}
