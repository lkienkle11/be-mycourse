// Error-related string literals and tightly coupled numeric limits (sentinels, shared messages).
// This file is the preferred place for those constants — see section headers per feature.
//
// # For AI agents / contributors
//
// Put here:
//   - String constants used in errors.New, fmt.Errorf, or other sentinels where the same text
//     must not be duplicated across packages.
//   - The **same** string when it must appear both in a sentinel (errors.New) **and** in the
//     default JSON `message` for an errcode — define it **once** in this file, then:
//   - reference it from pkg/errcode/messages.go (never duplicate the literal there), and
//   - reference it from pkg/media (or other) sentinel constructors.
//   - Small numeric limits that are defined together with those messages (same feature), e.g.
//     media upload max bytes next to the oversize message.
//
// Do not put here:
//   - The numeric HTTP JSON "code" enum — that stays in pkg/errcode/codes.go.
//
// The default message **map** (code int → string) stays in pkg/errcode/messages.go, but any
// entry whose text is shared with a constant in this file **must** use the constant from here
// so wording cannot drift (search this file before adding a parallel string in messages.go).
//
// Related: pkg/errors/upload_errors.go (ErrFileExceedsMaxUploadSize), pkg/errcode/messages.go (FileTooLarge).
package constants

// --- Media upload (multipart field "file", POST/PUT /api/v1/media/files) ---

// MaxMediaUploadFileBytes is the maximum allowed size in bytes for a single uploaded file part.
// Value is exactly 2 GiB: 2 × 1024 × 1024 × 1024.
const MaxMediaUploadFileBytes int64 = 2 * 1024 * 1024 * 1024

// MsgFileTooLargeUpload is the single canonical user-facing string for "uploaded file over 2 GiB cap".
// Used by:
//   - pkg/errcode defaultMessages[FileTooLarge] (JSON envelope message for code 2003),
//   - pkg/errors.ErrFileExceedsMaxUploadSize (errors.New / errors.Is sentinel).
//
// Do not copy this literal into messages.go or upload_errors.go — import constants.MsgFileTooLargeUpload.
const MsgFileTooLargeUpload = "Uploaded file exceeds the maximum allowed size (2 GiB per file)"

// Default JSON messages for media upstream errcodes 9010–9014 (pkg/errcode/messages.go references only).
const (
	MsgMediaB2BucketNotConfigured    = "Media B2 bucket is not configured (set MEDIA_B2_BUCKET / media.b2_bucket so URLs can use path <cdn>/<bucket>/<object>)"
	MsgMediaBunnyStreamNotConfigured = "Bunny Stream is not configured (library id and API key required)"
	MsgMediaBunnyCreateFailed        = "Bunny Stream failed to create video"
	MsgMediaBunnyUploadFailed        = "Bunny Stream failed to upload video content"
	MsgMediaBunnyInvalidResponse     = "Bunny Stream returned an invalid response"
	MsgMediaBunnyVideoNotFound       = "Bunny Stream video was not found"
	MsgMediaBunnyGetVideoFailed      = "Bunny Stream failed to get video details"
)
