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
) entities.UploadFileMetadata {
	base := entities.FileMetadata{
		Size:      sizeBytes,
		MimeType:  strings.TrimSpace(mimeType),
		Extension: utils.DetectExtension(filename, mimeType),
	}

	if kind == constants.FileKindVideo {
		duration := utils.FloatFromRaw(raw, "duration")
		if duration == 0 {
			duration = utils.FloatFromRaw(raw, "length")
		}
		fps := utils.FloatFromRaw(raw, "fps")
		if fps == 0 {
			fps = utils.FloatFromRaw(raw, "framerate")
		}
		return entities.UploadFileMetadata{
			SizeBytes:       base.Size,
			WidthBytes:      utils.IntFromRaw(raw, "width"),
			HeightBytes:     utils.IntFromRaw(raw, "height"),
			MimeType:        base.MimeType,
			Extension:       base.Extension,
			DurationSeconds: duration,
			Bitrate:         utils.IntFromRaw(raw, "bitrate"),
			FPS:             fps,
			VideoCodec:      utils.StringFromRaw(raw, "video_codec"),
			AudioCodec:      utils.StringFromRaw(raw, "audio_codec"),
			HasAudio:        parseBoolMetadata(raw, "has_audio"),
			IsHDR:           parseBoolMetadata(raw, "is_hdr"),
		}
	}
	w, h := utils.ImageSizeFromPayload(payload)
	return entities.UploadFileMetadata{
		SizeBytes:      base.Size,
		WidthBytes:     w,
		HeightBytes:    h,
		MimeType:       base.MimeType,
		Extension:      base.Extension,
		PageCount:      utils.IntFromRaw(raw, "page_count"),
		HasPassword:    parseBoolMetadata(raw, "has_password"),
		ArchiveEntries: utils.IntFromRaw(raw, "archive_entries"),
	}
}

func DefaultMediaProvider(kind constants.FileKind) constants.FileProvider {
	configured := strings.TrimSpace(setting.MediaSetting.AppMediaProvider)
	if configured != "" {
		return ResolveMediaProvider(kind, configured)
	}
	return ResolveMediaProvider(kind, "")
}

func parseBoolMetadata(raw entities.RawMetadata, key string) bool {
	v, ok := raw[key]
	if !ok {
		return false
	}
	switch typed := v.(type) {
	case bool:
		return typed
	default:
		parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprintf("%v", typed)))
		return err == nil && parsed
	}
}
