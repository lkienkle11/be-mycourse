package errors

import (
	stderrors "errors"

	"mycourse-io-be/internal/shared/constants"
)

// ErrInvalidProfileMediaFile is returned when image_file_id / avatar_file_id does not
// reference a READY non-video file suitable for raster image use (taxonomy cover, avatar).
var ErrInvalidProfileMediaFile = stderrors.New(constants.MsgInvalidProfileMediaFile)
