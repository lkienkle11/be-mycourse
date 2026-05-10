package media

import (
	"io"
	"mime/multipart"

	"mycourse-io-be/pkg/entities"
)

// CloseOpenedUploadParts closes all opened streams (best-effort).
func CloseOpenedUploadParts(parts []entities.OpenedUploadPart) {
	for _, p := range parts {
		if p.File != nil {
			_ = p.File.Close()
		}
	}
}

// OpenUploadParts opens each header; on error closes already-opened streams.
func OpenUploadParts(headers []*multipart.FileHeader) ([]entities.OpenedUploadPart, error) {
	parts := make([]entities.OpenedUploadPart, 0, len(headers))
	for _, h := range headers {
		f, err := h.Open()
		if err != nil {
			CloseOpenedUploadParts(parts)
			return nil, err
		}
		parts = append(parts, entities.OpenedUploadPart{File: f, Header: h})
	}
	return parts, nil
}

// DrainDiscard closes r after discarding any unread bytes (multipart cleanup helper).
func DrainDiscard(r io.Closer) {
	if r == nil {
		return
	}
	if rc, ok := r.(io.ReadCloser); ok {
		_, _ = io.Copy(io.Discard, rc)
	}
	_ = r.Close()
}
