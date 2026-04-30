package helper

import (
	"path/filepath"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/setting"
)

func ResolveMediaKind(kindRaw, mime, filename string) constants.FileKind {
	kind := constants.FileKind(strings.TrimSpace(kindRaw))
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

func ResolveMediaProvider(kind constants.FileKind, providerRaw string) constants.FileProvider {
	provider := constants.FileProvider(strings.TrimSpace(providerRaw))
	if provider != "" {
		return provider
	}
	if kind == constants.FileKindVideo {
		return constants.FileProviderBunny
	}
	return constants.FileProviderB2
}

func ResolveMediaKindFromServer(mime, filename string) (constants.FileKind, bool) {
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

func ResolveUploadProvider(kind constants.FileKind, kindInferred bool) constants.FileProvider {
	configured := strings.TrimSpace(setting.MediaSetting.AppMediaProvider)
	if configured != "" {
		return ResolveMediaProvider(kind, configured)
	}
	if !kindInferred {
		return constants.FileProviderLocal
	}
	return ResolveMediaProvider(kind, "")
}
