package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
)

func ParseMetadataJSON(raw string) (entities.RawMetadata, error) {
	if strings.TrimSpace(raw) == "" {
		return entities.RawMetadata{}, nil
	}
	out := make(entities.RawMetadata)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return NormalizeMetadata(out), nil
}

func NormalizeMetadata(in map[string]any) entities.RawMetadata {
	if in == nil {
		return entities.RawMetadata{}
	}
	out := make(entities.RawMetadata, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = v
	}
	return out
}

func ParseMetadataFromRaw(raw string) (entities.RawMetadata, error) {
	meta, err := ParseMetadataJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata json: %w", err)
	}
	return meta, nil
}

func BuildTypedMetadata(
	kind constants.FileKind,
	mimeType string,
	filename string,
	sizeBytes int64,
	payload []byte,
	raw entities.RawMetadata,
) any {
	base := entities.FileMetadata{
		Size:      sizeBytes,
		MimeType:  strings.TrimSpace(mimeType),
		Extension: detectExtension(filename, mimeType),
	}

	switch kind {
	case constants.FileKindVideo:
		width := intFromRaw(raw, "width")
		height := intFromRaw(raw, "height")
		if width <= 0 || height <= 0 {
			w, h := imageSizeFromPayload(payload)
			if width <= 0 {
				width = w
			}
			if height <= 0 {
				height = h
			}
		}
		return entities.VideoMetadata{
			FileMetadata:   base,
			Duration:       floatFromRaw(raw, "duration"),
			ThumbnailURL:   stringFromRaw(raw, "thumbnail_url"),
			BunnyVideoID:   nonEmpty(stringFromRaw(raw, "bunny_video_id"), stringFromRaw(raw, "video_guid")),
			BunnyLibraryID: stringFromRaw(raw, "bunny_library_id"),
			Size:           sizeBytes,
			Width:          width,
			Height:         height,
		}
	case constants.FileKindFile:
		w, h := imageSizeFromPayload(payload)
		if w > 0 && h > 0 {
			base.Width = w
			base.Height = h
			return entities.ImageMetadata{FileMetadata: base}
		}
		return entities.DocumentMetadata{
			FileMetadata: base,
			PageCount:    intFromRaw(raw, "page_count"),
		}
	default:
		return entities.DocumentMetadata{FileMetadata: base}
	}
}

func detectExtension(filename, mimeType string) string {
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

func imageSizeFromPayload(payload []byte) (int, int) {
	if len(payload) == 0 {
		return 0, 0
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func stringFromRaw(raw entities.RawMetadata, key string) string {
	if raw == nil {
		return ""
	}
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func intFromRaw(raw entities.RawMetadata, key string) int {
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

func floatFromRaw(raw entities.RawMetadata, key string) float64 {
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

func nonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		v := strings.TrimSpace(candidate)
		if v != "" {
			return v
		}
	}
	return ""
}
