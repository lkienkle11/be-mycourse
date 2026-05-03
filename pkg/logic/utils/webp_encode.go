//go:build cgo

// WebP encode via bimg (CGO + libvips).
// Active only when CGO_ENABLED=1 and libvips-dev is installed.
// Production build command: CGO_ENABLED=1 go build ...
package utils

import "github.com/h2non/bimg"

// EncodeWebP converts an image payload (JPEG, PNG, GIF, BMP, TIFF …) to WebP using bimg/libvips.
// Returns (encodedBytes, "image/webp", nil) on success.
//
// Always surround with AcquireEncodeGate / ReleaseEncodeGate so concurrent calls are bounded
// by constants.MaxConcurrentImageEncode.
func EncodeWebP(payload []byte) ([]byte, string, error) {
	converted, err := bimg.NewImage(payload).Convert(bimg.WEBP)
	if err != nil {
		return nil, "", err
	}
	return converted, "image/webp", nil
}
