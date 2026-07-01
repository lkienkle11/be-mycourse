package infra

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

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
	raw[domain.MediaMetaKeyMimeType] = typed.MimeType
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

func MergeMediaMetadataJSON(prevJSON []byte, overlay domain.RawMetadata) ([]byte, error) {
	base := map[string]any{}
	if len(prevJSON) > 0 {
		_ = json.Unmarshal(prevJSON, &base)
	}
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range overlay {
		base[k] = v
	}
	return json.Marshal(base)
}

// serializeMergedMetadataJSON marshals the merged raw metadata map into a
// JSON string that will be stored in the media_files.metadata_json JSONB
// column. Returns "{}" when there is nothing to persist or when marshalling
// fails (defensive: never break persistence on a serialisation error).
func serializeMergedMetadataJSON(merged domain.RawMetadata) string {
	if len(merged) == 0 {
		return "{}"
	}
	blob, err := json.Marshal(merged)
	if err != nil {
		return "{}"
	}
	return string(blob)
}

func mergeUploadInputMetadata(in domain.MediaUploadEntityInput) domain.RawMetadata {
	uploadedMeta := NormalizeMetadata(in.Uploaded.Metadata)
	merged := NormalizeMetadata(in.UploadedMeta)
	for k, v := range uploadedMeta {
		merged[k] = v
	}
	return merged
}

func streamMetadataFromMerged(merged domain.RawMetadata, typed domain.UploadFileMetadata) (
	bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider string,
	directPlayURL, hlsPlaylistURL, previewAnimationURL string,
	duration int64,
) {
	bunnyVideoID = utils.StringFromRaw(merged, "bunny_video_id")
	if bunnyVideoID == "" {
		bunnyVideoID = utils.StringFromRaw(merged, "video_guid")
	}
	videoID = utils.StringFromRaw(merged, domain.MediaMetaKeyVideoID)
	if videoID == "" {
		videoID = bunnyVideoID
	}
	thumbnailURL = SanitizeMetadataURL(utils.StringFromRaw(merged, domain.MediaMetaKeyThumbnailURL))
	embededHTML = SanitizeMetadataURL(utils.StringFromRaw(merged, domain.MediaMetaKeyEmbededHTML))
	bunnyLibraryID = utils.StringFromRaw(merged, "bunny_library_id")
	videoProvider = utils.StringFromRaw(merged, "video_provider")
	directPlayURL = SanitizeMetadataURL(utils.StringFromRaw(merged, domain.MediaMetaKeyDirectPlayURL))
	hlsPlaylistURL = SanitizeMetadataURL(utils.StringFromRaw(merged, domain.MediaMetaKeyHLSPlaylistURL))
	previewAnimationURL = SanitizeMetadataURL(utils.StringFromRaw(merged, domain.MediaMetaKeyPreviewAnimationURL))
	duration = int64(typed.DurationSeconds)
	if duration <= 0 {
		duration = int64(utils.FloatFromRaw(merged, "length"))
	}
	return bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider,
		directPlayURL, hlsPlaylistURL, previewAnimationURL, duration
}

func r2BucketFromUploadInput(in domain.MediaUploadEntityInput, merged domain.RawMetadata) string {
	bucket := strings.TrimSpace(in.R2Bucket)
	if bucket == "" {
		bucket = strings.TrimSpace(fmt.Sprintf("%v", merged["r2_bucket_name"]))
	}
	return bucket
}

func preservedOrNewEntityID(in domain.MediaUploadEntityInput) string {
	id := strings.TrimSpace(in.PreserveID)
	if in.GenerateNewID || id == "" {
		return uuid.NewString()
	}
	return id
}

func newFileEntityUploadCore(in domain.MediaUploadEntityInput, merged domain.RawMetadata, typed domain.UploadFileMetadata) *domain.File {
	return &domain.File{
		ID:           preservedOrNewEntityID(in),
		Kind:         in.Kind,
		Provider:     in.Provider,
		Filename:     in.Filename,
		MimeType:     in.ContentType,
		SizeBytes:    in.SizeBytes,
		URL:          in.Uploaded.URL,
		OriginURL:    in.Uploaded.OriginURL,
		ObjectKey:    in.Uploaded.ObjectKey,
		Status:       constants.FileStatusReady,
		R2BucketName: r2BucketFromUploadInput(in, merged),
		Metadata:     typed,
		// Persist the merged provider metadata into the JSONB column.
		// Without this assignment the database row would always store "{}"
		// (see fileToRow in repos.go) even though Bunny/B2 return useful
		// fields like length, framerate, resolution, thumbnail_filename, ...
		MetadataJSON:       serializeMergedMetadataJSON(merged),
		CreatedAt:          in.CreatedAt,
		UpdatedAt:          in.UpdatedAt,
		RowVersion:         1,
		ContentFingerprint: "",
	}
}

type uploadStreamFields struct {
	bunnyVideoID        string
	videoID             string
	thumbnailURL        string
	embededHTML         string
	bunnyLibraryID      string
	videoProvider       string
	directPlayURL       string
	hlsPlaylistURL      string
	previewAnimationURL string
	duration            int64
}

func attachStreamFieldsToFile(f *domain.File, stream uploadStreamFields) {
	f.BunnyVideoID = stream.bunnyVideoID
	f.BunnyLibraryID = stream.bunnyLibraryID
	f.VideoID = stream.videoID
	f.ThumbnailURL = stream.thumbnailURL
	f.EmbededHTML = stream.embededHTML
	f.DirectPlayURL = stream.directPlayURL
	f.HLSPlaylistURL = stream.hlsPlaylistURL
	f.PreviewAnimationURL = stream.previewAnimationURL
	f.Duration = stream.duration
	f.VideoProvider = stream.videoProvider
}

func fileEntityFromUploadStreamFields(
	in domain.MediaUploadEntityInput,
	merged domain.RawMetadata,
	typed domain.UploadFileMetadata,
	stream uploadStreamFields,
) *domain.File {
	f := newFileEntityUploadCore(in, merged, typed)
	attachStreamFieldsToFile(f, stream)
	return f
}

func BuildMediaFileEntityFromUpload(in domain.MediaUploadEntityInput) *domain.File {
	merged := mergeUploadInputMetadata(in)
	typed := BuildTypedMetadata(in.Kind, in.ContentType, in.Filename, in.SizeBytes, in.Payload, merged)
	ApplyTypedMetadataToRaw(merged, typed)
	bv, vid, thumb, embed, lib, vprov, direct, hls, preview, dur := streamMetadataFromMerged(merged, typed)
	return fileEntityFromUploadStreamFields(in, merged, typed, uploadStreamFields{
		bunnyVideoID: bv, videoID: vid, thumbnailURL: thumb, embededHTML: embed,
		bunnyLibraryID: lib, videoProvider: vprov,
		directPlayURL: direct, hlsPlaylistURL: hls, previewAnimationURL: preview,
		duration: dur,
	})
}

// ResolveMediaUploadObjectKey selects the object key before upload (explicit key, Bunny empty until GUID, local nano key, R2 eight-digit prefix key).
func ResolveMediaUploadObjectKey(reqObjectKey, filename string, provider string) string {
	if dk := strings.TrimSpace(reqObjectKey); dk != "" {
		return strings.TrimLeft(dk, "/")
	}
	switch provider {
	case constants.FileProviderBunny:
		return ""
	case constants.FileProviderLocal:
		return buildLocalUploadObjectKey("", filename)
	default:
		return BuildObjectStorageKey(filename)
	}
}

func buildLocalUploadObjectKey(defaultKey, filename string) string {
	key := strings.TrimSpace(defaultKey)
	if key != "" {
		return strings.TrimLeft(key, "/")
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = strings.ReplaceAll(strings.TrimSpace(base), " ", "-")
	if base == "" {
		base = "file"
	}
	return fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), base, ext)
}

func sanitizeObjectStorageUploadBase(filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = strings.TrimSpace(base)
	if base == "" {
		base = "file"
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), ".-_")
	if s == "" {
		s = "file"
	}
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s + strings.ToLower(ext)
}

// BuildObjectStorageKey builds default R2 object keys: eight random digits, hyphen, sanitized filename.
func BuildObjectStorageKey(filename string) string {
	return utils.GenerateRandomDigits(8) + "-" + sanitizeObjectStorageUploadBase(filename)
}
