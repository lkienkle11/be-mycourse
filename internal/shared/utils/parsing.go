package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	_ "golang.org/x/image/webp"

	"mycourse-io-be/internal/shared/parsebool"
)

func DetectExtension(filename, mimeType string) string {
	name := strings.TrimSpace(filename)
	if dot := strings.LastIndex(name, "."); dot >= 0 && dot < len(name)-1 {
		return strings.ToLower(name[dot+1:])
	}
	mime := strings.TrimSpace(mimeType)
	if slash := strings.LastIndex(mime, "/"); slash >= 0 && slash < len(mime)-1 {
		return strings.ToLower(mime[slash+1:])
	}
	return ""
}

func ImageSizeFromPayload(payload []byte) (int, int) {
	if len(payload) == 0 {
		return 0, 0
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func StringFromRaw(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func IntFromRaw(raw map[string]any, key string) int {
	if raw == nil {
		return 0
	}
	switch v := raw[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func FloatFromRaw(raw map[string]any, key string) float64 {
	if raw == nil {
		return 0
	}
	switch v := raw[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func NonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		v := strings.TrimSpace(candidate)
		if v != "" {
			return v
		}
	}
	return ""
}

// ParseBoolLoose delegates to parsebool.Loose (env, YAML, multipart form fields).
func ParseBoolLoose(s string) bool {
	return parsebool.Loose(s)
}

// ContentFingerprint returns SHA-256 hex of payload (e.g. media skip-upload dedupe).
func ContentFingerprint(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
