package helper

import (
	"encoding/json"
	"fmt"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/setting"
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
		Extension: utils.DetectExtension(filename, mimeType),
	}

	switch kind {
	case constants.FileKindVideo:
		width := utils.IntFromRaw(raw, "width")
		height := utils.IntFromRaw(raw, "height")
		if width <= 0 || height <= 0 {
			w, h := utils.ImageSizeFromPayload(payload)
			if width <= 0 {
				width = w
			}
			if height <= 0 {
				height = h
			}
		}
		return entities.VideoMetadata{
			FileMetadata:   base,
			Duration:       utils.FloatFromRaw(raw, "duration"),
			ThumbnailURL:   utils.StringFromRaw(raw, "thumbnail_url"),
			BunnyVideoID:   utils.NonEmpty(utils.StringFromRaw(raw, "bunny_video_id"), utils.StringFromRaw(raw, "video_guid")),
			BunnyLibraryID: utils.StringFromRaw(raw, "bunny_library_id"),
			VideoProvider:  utils.StringFromRaw(raw, "video_provider"),
			Size:           sizeBytes,
			Width:          width,
			Height:         height,
		}
	case constants.FileKindFile:
		w, h := utils.ImageSizeFromPayload(payload)
		if w > 0 && h > 0 {
			base.Width = w
			base.Height = h
			return entities.ImageMetadata{FileMetadata: base}
		}
		return entities.DocumentMetadata{
			FileMetadata: base,
			PageCount:    utils.IntFromRaw(raw, "page_count"),
		}
	default:
		return entities.DocumentMetadata{FileMetadata: base}
	}
}

func DefaultMediaProvider(kind constants.FileKind) constants.FileProvider {
	configured := strings.TrimSpace(setting.MediaSetting.AppMediaProvider)
	if configured != "" {
		return ResolveMediaProvider(kind, configured)
	}
	return ResolveMediaProvider(kind, "")
}
