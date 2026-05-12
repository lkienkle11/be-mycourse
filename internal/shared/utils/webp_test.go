package utils_test

// Regression tests for ImageSizeFromPayload.
//
// Root-cause fix: parsing.go blank-imports
// _ "golang.org/x/image/webp" so that image.DecodeConfig can decode WebP
// payloads produced by the bimg/CGO encode pipeline. Without that
// import, image.DecodeConfig returned (0, 0) for every WebP payload.

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"mycourse-io-be/internal/shared/utils"
)

const imgW, imgH = 16, 16

func makeTestRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, white)
		}
	}
	return img
}

func testPNGBytes(w, h int) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, makeTestRGBA(w, h)); err != nil {
		panic("testPNGBytes: " + err.Error())
	}
	return buf.Bytes()
}

func testJPEGBytes(w, h int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, makeTestRGBA(w, h), &jpeg.Options{Quality: 80}); err != nil {
		panic("testJPEGBytes: " + err.Error())
	}
	return buf.Bytes()
}

func testGIFBytes(w, h int) []byte {
	palette := color.Palette{color.White, color.Black}
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		panic("testGIFBytes: " + err.Error())
	}
	return buf.Bytes()
}

// TestImageSizeFromPayload_StandardFormats covers PNG, JPEG, GIF, empty and
// corrupted payloads using a table-driven style.
func TestImageSizeFromPayload_StandardFormats(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
		wantW   int
		wantH   int
	}{
		{"PNG", testPNGBytes(imgW, imgH), imgW, imgH},
		{"JPEG", testJPEGBytes(imgW, imgH), imgW, imgH},
		{"GIF", testGIFBytes(imgW, imgH), imgW, imgH},
		{"nil_payload", nil, 0, 0},
		{"empty_slice", []byte{}, 0, 0},
		{"corrupted_bytes", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xAB, 0xCD, 0xDE, 0xAD, 0xBE, 0xEF}, 0, 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, h := utils.ImageSizeFromPayload(tc.payload)
			if w != tc.wantW || h != tc.wantH {
				t.Errorf("%s: ImageSizeFromPayload got (%d,%d), want (%d,%d)", tc.name, w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

// TestImageSizeFromPayload_WebP_Regression is the primary regression test:
// the upload pipeline encodes images to WebP via bimg before storing.
// Before the fix, image.DecodeConfig had no WebP decoder registered, so width
// and height were always 0 for WebP payloads.
//
// When CGO / libvips is unavailable, EncodeWebP returns
// ErrImageEncodeBusy and the test is skipped — it passes on all CGO-enabled
// builds (production + CI).
func TestImageSizeFromPayload_WebP_Regression(t *testing.T) {
	src := testPNGBytes(imgW, imgH)
	webpBytes, _, err := utils.EncodeWebP(src)
	if err != nil {
		t.Skipf("CGO/libvips not available; skipping WebP regression: %v", err)
	}
	w, h := utils.ImageSizeFromPayload(webpBytes)
	if w <= 0 || h <= 0 {
		t.Errorf("WebP regression: got (%d,%d), want (>0,>0). "+
			"Ensure _ \"golang.org/x/image/webp\" is imported in parsing.go.", w, h)
	}
}
