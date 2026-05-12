//go:build !cgo

// Stub — active when CGO_ENABLED=0 (CI default, local code review, pure-Go builds).
// EncodeWebP always returns ErrImageEncodeBusy (errcode 9017).
// Build with CGO_ENABLED=1 and libvips-dev to enable actual WebP conversion.
package utils

import apperrors "mycourse-io-be/internal/shared/errors"

// EncodeWebP is disabled because CGO is not enabled in this build.
// The caller receives ErrImageEncodeBusy (errcode 9017 → HTTP 503).
func EncodeWebP(_ []byte) ([]byte, string, error) {
	return nil, "", apperrors.ErrImageEncodeBusy
}
