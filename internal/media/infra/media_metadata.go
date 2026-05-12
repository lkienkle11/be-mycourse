package infra

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

func ParseMetadataJSON(raw string) (domain.RawMetadata, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.RawMetadata{}, nil
	}
	out := make(domain.RawMetadata)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return NormalizeMetadata(out), nil
}

func NormalizeMetadata(in map[string]any) domain.RawMetadata {
	if in == nil {
		return domain.RawMetadata{}
	}
	out := make(domain.RawMetadata, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = v
	}
	return out
}

func ParseMetadataFromRaw(raw string) (domain.RawMetadata, error) {
	meta, err := ParseMetadataJSON(raw)
	if err != nil {
		return nil, fmt.Errorf(constants.MsgInvalidMetadataJSON, err)
	}
	return meta, nil
}

func typedMetadataBase(mimeType, filename string, sizeBytes int64) domain.FileMetadata {
	return domain.FileMetadata{
		Size:      sizeBytes,
		MimeType:  strings.TrimSpace(mimeType),
		Extension: utils.DetectExtension(filename, mimeType),
	}
}

func buildVideoTypedMetadata(base domain.FileMetadata, raw domain.RawMetadata) domain.UploadFileMetadata {
	duration := utils.FloatFromRaw(raw, "duration_seconds")
	if duration == 0 {
		duration = utils.FloatFromRaw(raw, "duration")
	}
	if duration == 0 {
		duration = utils.FloatFromRaw(raw, "length")
	}
	fps := utils.FloatFromRaw(raw, "fps")
	if fps == 0 {
		fps = utils.FloatFromRaw(raw, "framerate")
	}
	width := utils.IntFromRaw(raw, "width_bytes")
	if width == 0 {
		width = utils.IntFromRaw(raw, "width")
	}
	height := utils.IntFromRaw(raw, "height_bytes")
	if height == 0 {
		height = utils.IntFromRaw(raw, "height")
	}
	return domain.UploadFileMetadata{
		SizeBytes:       base.Size,
		WidthBytes:      width,
		HeightBytes:     height,
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

func buildImageTypedMetadata(base domain.FileMetadata, raw domain.RawMetadata, payload []byte) domain.UploadFileMetadata {
	w, h := utils.ImageSizeFromPayload(payload)
	// When payload is nil (read-back from DB via ToMediaEntity) or the decoder
	// cannot parse it, fall back to the persisted width/height stored in the
	// raw JSON metadata ("width"/"height" keys written by uploadMetadataToRaw).
	if w == 0 {
		w = utils.IntFromRaw(raw, "width_bytes")
	}
	if w == 0 {
		w = utils.IntFromRaw(raw, "width")
	}
	if h == 0 {
		h = utils.IntFromRaw(raw, "height_bytes")
	}
	if h == 0 {
		h = utils.IntFromRaw(raw, "height")
	}
	return domain.UploadFileMetadata{
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

func BuildTypedMetadata(
	kind string,
	mimeType string,
	filename string,
	sizeBytes int64,
	payload []byte,
	raw domain.RawMetadata,
) domain.UploadFileMetadata {
	base := typedMetadataBase(mimeType, filename, sizeBytes)
	if kind == constants.FileKindVideo {
		return buildVideoTypedMetadata(base, raw)
	}
	return buildImageTypedMetadata(base, raw, payload)
}

// ApplyTypedMetadataToRaw writes the public UploadFileMetadata shape into the
// raw JSONB metadata map. Provider-specific fields (for example Bunny's
// available_resolutions or thumbnail_blurhash) remain in the same map, but
// these stable keys guarantee metadata_json can always rehydrate the API
// `metadata` response exactly.
func ApplyTypedMetadataToRaw(raw domain.RawMetadata, typed domain.UploadFileMetadata) {
	if raw == nil {
		return
	}
	raw["size_bytes"] = typed.SizeBytes
	raw["width_bytes"] = typed.WidthBytes
	raw["height_bytes"] = typed.HeightBytes
	raw["mime_type"] = typed.MimeType
	raw["extension"] = typed.Extension
	raw["duration_seconds"] = typed.DurationSeconds
	raw["bitrate"] = typed.Bitrate
	raw["fps"] = typed.FPS
	raw["video_codec"] = typed.VideoCodec
	raw["audio_codec"] = typed.AudioCodec
	raw["has_audio"] = typed.HasAudio
	raw["is_hdr"] = typed.IsHDR
	raw["page_count"] = typed.PageCount
	raw["has_password"] = typed.HasPassword
	raw["archive_entries"] = typed.ArchiveEntries
}

func DefaultMediaProvider(kind string) string {
	configured := strings.TrimSpace(setting.MediaSetting.AppMediaProvider)
	if configured != "" {
		return ResolveMediaProvider(kind, configured)
	}
	return ResolveMediaProvider(kind, "")
}

func parseBoolMetadata(raw domain.RawMetadata, key string) bool {
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
