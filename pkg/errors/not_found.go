package errors

import (
	stderrors "errors"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
)

// ErrNotFound means the requested persisted row or aggregate does not exist.
// Handlers in api/ should compare with this sentinel instead of importing gorm.io/gorm.
var ErrNotFound = stderrors.New(constants.MsgNotFound)

// MapRecordNotFound translates gorm.ErrRecordNotFound into ErrNotFound.
func MapRecordNotFound(err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
