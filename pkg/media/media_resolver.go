package media

import (
	"fmt"
	"html"
	"path/filepath"
	"strconv"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/setting"
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

// EnrichBunnyVideoDetail normalizes thumbnail URL from alternate JSON fields returned by Bunny.
func EnrichBunnyVideoDetail(d *entities.BunnyVideoDetail) {
	if d == nil {
		return
	}
	if strings.TrimSpace(d.ThumbnailURL) == "" {
		d.ThumbnailURL = strings.TrimSpace(d.DefaultThumbnailURL)
	}
}

// EffectiveBunnyThumbnailURL returns the best-known thumbnail URL after enrichment.
func EffectiveBunnyThumbnailURL(d *entities.BunnyVideoDetail) string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(d.ThumbnailURL)
}

// FormatBunnyVideoIDString prefers Bunny’s numeric id when present; otherwise returns guid.
func FormatBunnyVideoIDString(d *entities.BunnyVideoDetail) string {
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

// applyBunnyVideoTelemetry writes non-zero video dimension and codec fields from d into meta.
// It mirrors the keys used by patchBunnyWebhookMetadataJSON so both the upload path and the
// webhook path store identical key names.
func applyBunnyVideoTelemetry(meta entities.RawMetadata, d *entities.BunnyVideoDetail) {
	telemetry := map[string]any{
		"width":       d.Width,
		"height":      d.Height,
		"length":      d.Length,
		"framerate":   d.Framerate,
		"bitrate":     d.Bitrate,
		"video_codec": strings.TrimSpace(d.VideoCodec),
		"audio_codec": strings.TrimSpace(d.AudioCodec),
	}
	for k, v := range telemetry {
		switch val := v.(type) {
		case int:
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
		}
	}
}

// ApplyBunnyDetailToMetadata writes Bunny Stream video detail fields into upload/raw metadata.
// It populates video telemetry (width, height, length, framerate, bitrate, codecs) alongside
// video_id, thumbnail_url, and embeded_html. Only non-zero/non-empty values are written so
// callers may safely merge into an already-populated metadata map.
func ApplyBunnyDetailToMetadata(meta entities.RawMetadata, d *entities.BunnyVideoDetail, libraryID, streamPlayBase string) {
	if meta == nil || d == nil {
		return
	}
	EnrichBunnyVideoDetail(d)
	applyBunnyVideoTelemetry(meta, d)
	if vid := FormatBunnyVideoIDString(d); vid != "" {
		meta[constants.MediaMetaKeyVideoID] = vid
	}
	if thumb := EffectiveBunnyThumbnailURL(d); thumb != "" {
		meta[constants.MediaMetaKeyThumbnailURL] = thumb
	}
	if embed := ResolveBunnyEmbedHTML(libraryID, d.GUID, streamPlayBase); embed != "" {
		meta[constants.MediaMetaKeyEmbededHTML] = embed
	}
}
