package helper

import (
	"path/filepath"
	"strings"

	"mycourse-io-be/constants"
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
