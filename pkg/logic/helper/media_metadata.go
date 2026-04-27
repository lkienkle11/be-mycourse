package helper

import (
	"encoding/json"
	"fmt"
	"strconv"
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
		duration := utils.FloatFromRaw(raw, "duration")
		if duration == 0 {
			duration = utils.FloatFromRaw(raw, "length")
		}
		fps := utils.FloatFromRaw(raw, "fps")
		if fps == 0 {
			fps = utils.FloatFromRaw(raw, "framerate")
		}
		hasAudio := false
		if v, ok := raw["has_audio"]; ok {
			switch typed := v.(type) {
			case bool:
				hasAudio = typed
			default:
				parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprintf("%v", typed)))
				hasAudio = err == nil && parsed
			}
		}
		isHDR := false
		if v, ok := raw["is_hdr"]; ok {
			switch typed := v.(type) {
			case bool:
				isHDR = typed
			default:
				parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprintf("%v", typed)))
				isHDR = err == nil && parsed
			}
		}
		return entities.VideoMetadata{
			FileMetadata: base,
			Width:        utils.IntFromRaw(raw, "width"),
			Height:       utils.IntFromRaw(raw, "height"),
			Duration:     duration,
			Bitrate:      utils.IntFromRaw(raw, "bitrate"),
			FPS:          fps,
			VideoCodec:   utils.StringFromRaw(raw, "video_codec"),
			AudioCodec:   utils.StringFromRaw(raw, "audio_codec"),
			HasAudio:     hasAudio,
			IsHDR:        isHDR,
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
