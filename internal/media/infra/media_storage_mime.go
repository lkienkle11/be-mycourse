package infra

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"

	"mycourse-io-be/internal/shared/constants"
)

var blockedStorageMIMETypes = map[string]struct{}{
	"text/html":              {},
	"application/xhtml+xml":  {},
	"application/javascript": {},
	"text/javascript":        {},
	"image/svg+xml":          {},
}

var ooxmlExtensions = map[string]struct{}{
	".docx": {},
	".xlsx": {},
	".pptx": {},
}

// MIMEForUploadRouting returns content-detected MIME for kind/provider/image-encode
// routing. Client multipart Content-Type is ignored. Returns "" when detection is
// empty or blocked so callers fall back to filename extension only.
func MIMEForUploadRouting(payload []byte, filename, clientMIME string) string {
	_ = clientMIME
	detected := detectTrustedMIME(payload, filename)
	if detected != "" && !isBlockedStorageMIME(detected) {
		return detected
	}
	return ""
}

// CanonicalStorageMIME derives a server-trusted MIME for DB persistence and R2
// ContentType. clientMIME from multipart headers is ignored.
func CanonicalStorageMIME(payload []byte, filename, clientMIME, kind string) string {
	_ = clientMIME
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == ".webp" {
		if isWebPPayload(payload) {
			return "image/webp"
		}
		return "application/octet-stream"
	}
	detected := detectTrustedMIME(payload, filename)
	return applyStorageMIMEPolicy(detected, filename, kind)
}

func detectTrustedMIME(payload []byte, filename string) string {
	if len(payload) == 0 {
		return ""
	}
	_ = filename
	mtype := mimetype.Detect(payload)
	if mtype == nil {
		return ""
	}
	return baseMIME(mtype.String())
}

func applyStorageMIMEPolicy(detected, filename, kind string) string {
	detected = baseMIME(detected)
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))

	if isBlockedStorageMIME(detected) {
		return "application/octet-stream"
	}

	if detected == "application/zip" {
		if _, isOOXML := ooxmlExtensions[ext]; isOOXML {
			return "application/octet-stream"
		}
	}

	if strings.TrimSpace(kind) == constants.FileKindVideo {
		if strings.HasPrefix(detected, "video/") {
			return detected
		}
		return "application/octet-stream"
	}

	if detected == "" || detected == "application/octet-stream" {
		return "application/octet-stream"
	}

	if policyMismatch(detected, ext) {
		return "application/octet-stream"
	}

	return detected
}

func policyMismatch(detected, ext string) bool {
	switch ext {
	case ".pdf":
		return detected != "application/pdf"
	case ".txt":
		return !strings.HasPrefix(detected, "text/plain")
	case ".zip":
		return detected != "application/zip"
	default:
		return false
	}
}

func baseMIME(mimeType string) string {
	mimeType = strings.TrimSpace(mimeType)
	if i := strings.Index(mimeType, ";"); i >= 0 {
		return strings.TrimSpace(mimeType[:i])
	}
	return mimeType
}

func isWebPPayload(payload []byte) bool {
	return len(payload) >= 12 &&
		bytes.Equal(payload[0:4], []byte("RIFF")) &&
		bytes.Equal(payload[8:12], []byte("WEBP"))
}

func isBlockedStorageMIME(mimeType string) bool {
	_, blocked := blockedStorageMIMETypes[strings.ToLower(strings.TrimSpace(mimeType))]
	return blocked
}
