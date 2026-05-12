package infra

import (
	"fmt"
	"html"
	"path/filepath"
	"strconv"
	"strings"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/setting"
)

func ResolveMediaKind(kindRaw, mime, filename string) string {
	kind := string(strings.TrimSpace(kindRaw))
	if kind == constants.FileKindFile || kind == constants.FileKindVideo {
		return kind
	}
	if strings.HasPrefix(strings.ToLower(mime), "video/") {
		return constants.FileKindVideo
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".mov", ".mkv", ".avi", ".webm":
		return constants.FileKindVideo
	default:
		return constants.FileKindFile
	}
}

func ResolveMediaProvider(kind string, providerRaw string) string {
	provider := string(strings.TrimSpace(providerRaw))
	if provider != "" {
		return provider
	}
	if kind == constants.FileKindVideo {
		return constants.FileProviderBunny
	}
	return constants.FileProviderB2
}

func ResolveMediaKindFromServer(mime, filename string) (string, bool) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "video/") {
		return constants.FileKindVideo, true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".mov", ".mkv", ".avi", ".webm":
		return constants.FileKindVideo, true
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".pdf", ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx", ".txt", ".zip", ".rar", ".7z", ".tar", ".gz":
		return constants.FileKindFile, true
	default:
		return constants.FileKindFile, false
	}
}

// IsImageMIMEOrExt reports whether a file should be treated as an image based on its MIME type or
// file extension. Used to decide whether WebP conversion should be applied before upload.
func IsImageMIMEOrExt(mime, filename string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".webp":
		return true
	}
	return false
}

func ResolveUploadProvider(kind string, kindInferred bool) string {
	configured := strings.TrimSpace(setting.MediaSetting.AppMediaProvider)
	if configured != "" {
		return ResolveMediaProvider(kind, configured)
	}
	if !kindInferred {
		return constants.FileProviderLocal
	}
	return ResolveMediaProvider(kind, "")
}

// EnrichBunnyVideoDetail normalises the thumbnail URL on the detail struct.
//
// Bunny's get-video response does NOT include a fully qualified thumbnail URL.
// It only returns `thumbnailFileName` (e.g. "thumbnail.jpg") and the caller is
// expected to combine that with their CDN pull-zone hostname:
//
//	https://{cdn_hostname}/{guid}/{thumbnailFileName}
//
// We accept three sources, in priority order:
//  1. `thumbnailUrl` (legacy / forward-compatible; rarely populated by Bunny).
//  2. `defaultThumbnailUrl` (legacy alias).
//  3. CDN hostname + `thumbnailFileName` (canonical for current Bunny API).
func EnrichBunnyVideoDetail(d *domain.BunnyVideoDetail) {
	if d == nil {
		return
	}
	if strings.TrimSpace(d.ThumbnailURL) != "" {
		return
	}
	if alt := strings.TrimSpace(d.DefaultThumbnailURL); alt != "" {
		d.ThumbnailURL = alt
		return
	}
	d.ThumbnailURL = buildBunnyThumbnailFromCDN(d)
}

// buildBunnyThumbnailFromCDN composes the thumbnail URL from the configured
// CDN pull-zone hostname and the video's thumbnail filename. Returns "" when
// either piece is missing.
func buildBunnyThumbnailFromCDN(d *domain.BunnyVideoDetail) string {
	host := strings.TrimSpace(setting.MediaSetting.BunnyStreamCDNHostname)
	file := strings.TrimSpace(d.ThumbnailFileName)
	guid := strings.TrimSpace(d.GUID)
	if host == "" || file == "" || guid == "" {
		return ""
	}
	base := host
	if !strings.HasPrefix(strings.ToLower(host), "http://") &&
		!strings.HasPrefix(strings.ToLower(host), "https://") {
		base = "https://" + host
	}
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s/%s", base, guid, file)
}

// EffectiveBunnyThumbnailURL returns the best-known thumbnail URL after enrichment.
func EffectiveBunnyThumbnailURL(d *domain.BunnyVideoDetail) string {
	if d == nil {
		return ""
	}
	EnrichBunnyVideoDetail(d)
	return strings.TrimSpace(d.ThumbnailURL)
}

// FormatBunnyVideoIDString prefers Bunny’s numeric id when present; otherwise returns guid.
func FormatBunnyVideoIDString(d *domain.BunnyVideoDetail) string {
	if d == nil {
		return ""
	}
	if d.BunnyNumericID > 0 {
		return strconv.FormatInt(d.BunnyNumericID, 10)
	}
	return strings.TrimSpace(d.GUID)
}

// ResolveBunnyEmbedURL builds the iframe embed URL (…/embed/{libraryId}/{guid}) from the configured play base.
func ResolveBunnyEmbedURL(libraryID, videoGUID, streamPlayBase string) string {
	lib := strings.TrimSpace(libraryID)
	guid := strings.TrimSpace(videoGUID)
	if lib == "" || guid == "" {
		return ""
	}
	base := strings.TrimSpace(streamPlayBase)
	if base == "" {
		base = "https://iframe.mediadelivery.net/play"
	}
	base = strings.TrimRight(base, "/")
	base = strings.TrimSuffix(base, "/play")
	embedBase := base + "/embed"
	return fmt.Sprintf("%s/%s/%s", embedBase, lib, guid)
}

// ResolveBunnyEmbedHTML returns a minimal iframe snippet for embedding the Bunny player.
func ResolveBunnyEmbedHTML(libraryID, videoGUID, streamPlayBase string) string {
	src := ResolveBunnyEmbedURL(libraryID, videoGUID, streamPlayBase)
	if src == "" {
		return ""
	}
	esc := html.EscapeString(src)
	return fmt.Sprintf(
		`<iframe src="%s" loading="lazy" width="100%%" height="100%%" style="border:0" allow="accelerometer; gyroscope; autoplay; encrypted-media; picture-in-picture;" allowfullscreen></iframe>`,
		esc,
	)
}

// applyBunnyVideoTelemetry writes Bunny detail fields into meta using snake_case keys.
//
// Only non-zero / non-empty values are written so the helper is safe to call
// multiple times (e.g. once after upload, again from the webhook) without
// destroying previously-populated keys.
//
// The codec column prefers `outputCodecs` (current Bunny API field name) and
// falls back to the legacy `videoCodec` field for forward compatibility.
func applyBunnyVideoTelemetry(meta domain.RawMetadata, d *domain.BunnyVideoDetail) {
	videoCodec := strings.TrimSpace(d.OutputCodecs)
	if videoCodec == "" {
		videoCodec = strings.TrimSpace(d.VideoCodec)
	}
	telemetry := map[string]any{
		// Core video telemetry
		"width":      d.Width,
		"height":     d.Height,
		"length":     d.Length,
		"framerate":  d.Framerate,
		"bitrate":    d.Bitrate,
		"rotation":   d.Rotation,
		"video_codec": videoCodec,
		"audio_codec": strings.TrimSpace(d.AudioCodec),
		"output_codecs": strings.TrimSpace(d.OutputCodecs),

		// Encoding / availability
		"available_resolutions":    strings.TrimSpace(d.AvailableResolutions),
		"encode_progress":          d.EncodeProgress,
		"storage_size":             d.StorageSize,
		"has_mp4_fallback":         d.HasMP4Fallback,
		"has_original":             d.HasOriginal,
		"has_high_quality_preview": d.HasHighQualityPreview,
		"jit_encoding_enabled":     d.JitEncodingEnabled,

		// Thumbnail bits (the resolved thumbnail_url is written separately
		// via ApplyBunnyDetailToMetadata so it is consistent with the
		// File.ThumbnailURL column).
		"thumbnail_filename":  strings.TrimSpace(d.ThumbnailFileName),
		"thumbnail_blurhash": strings.TrimSpace(d.ThumbnailBlurhash),
		"thumbnail_count":    d.ThumbnailCount,

		// Descriptive metadata
		"title":         strings.TrimSpace(d.Title),
		"description":   strings.TrimSpace(d.Description),
		"date_uploaded": strings.TrimSpace(d.DateUploaded),
		"views":         d.Views,
		"is_public":     d.IsPublic,
		"category":      strings.TrimSpace(d.Category),
		"collection_id": strings.TrimSpace(d.CollectionID),
		"original_hash": strings.TrimSpace(d.OriginalHash),
	}
	for k, v := range telemetry {
		writeNonZeroMetadata(meta, k, v)
	}
}

// writeNonZeroMetadata writes v into meta[k] only when v carries useful data
// (positive numbers, non-empty strings, or `true` booleans).
func writeNonZeroMetadata(meta domain.RawMetadata, k string, v any) {
	switch val := v.(type) {
	case int:
		if val > 0 {
			meta[k] = val
		}
	case int64:
		if val > 0 {
			meta[k] = val
		}
	case float64:
		if val > 0 {
			meta[k] = val
		}
	case string:
		if val != "" {
			meta[k] = val
		}
	case bool:
		if val {
			meta[k] = val
		}
	}
}

// ApplyBunnyDetailToMetadata writes Bunny Stream video detail fields into upload/raw metadata.
// It populates video telemetry (width, height, length, framerate, bitrate, codecs) alongside
// video_id, thumbnail_url, and embeded_html. Only non-zero/non-empty values are written so
// callers may safely merge into an already-populated metadata map.
func ApplyBunnyDetailToMetadata(meta domain.RawMetadata, d *domain.BunnyVideoDetail, libraryID, streamPlayBase string) {
	if meta == nil || d == nil {
		return
	}
	EnrichBunnyVideoDetail(d)
	applyBunnyVideoTelemetry(meta, d)
	if vid := FormatBunnyVideoIDString(d); vid != "" {
		meta[domain.MediaMetaKeyVideoID] = vid
	}
	if thumb := EffectiveBunnyThumbnailURL(d); thumb != "" {
		meta[domain.MediaMetaKeyThumbnailURL] = thumb
	}
	if embed := ResolveBunnyEmbedHTML(libraryID, d.GUID, streamPlayBase); embed != "" {
		meta[domain.MediaMetaKeyEmbededHTML] = embed
	}
}
