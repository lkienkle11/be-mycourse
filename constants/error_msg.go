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

// --- Media upload (multipart fields "files" / legacy "file", POST/PUT /api/v1/media/files) ---

// MaxMediaUploadFileBytes is the maximum allowed size in bytes for a single uploaded file part.
// Value is exactly 2 GiB: 2 × 1024 × 1024 × 1024.
const MaxMediaUploadFileBytes int64 = 2 * 1024 * 1024 * 1024

// MaxMediaMultipartTotalBytes is the maximum combined size in bytes for all file parts in one
// multipart create/update request (sum of buffered payloads). Numerically equal to MaxMediaUploadFileBytes
// but applies to the aggregate across parts (AND each part must still satisfy MaxMediaUploadFileBytes).
const MaxMediaMultipartTotalBytes int64 = 2 * 1024 * 1024 * 1024

// MaxMediaFilesPerRequest is the maximum number of file parts allowed in one POST /media/files or
// PUT /media/files/:id multipart request (field "files", or legacy "file").
const MaxMediaFilesPerRequest = 5

// MaxMediaBatchDelete is the maximum number of object keys in one batch delete request.
const MaxMediaBatchDelete = 10

// MaxConcurrentMediaUploadWorkers bounds parallel cloud uploads within one multi-file create/update.
const MaxConcurrentMediaUploadWorkers = 5

// MediaMultipartParseMemoryBytes is passed to http.Request.ParseMultipartForm (memory before spill to disk).
const MediaMultipartParseMemoryBytes int64 = 64 << 20

// MsgFileTooLargeUpload is the single canonical user-facing string for "uploaded file over 2 GiB cap".
// Used by:
//   - pkg/errcode defaultMessages[FileTooLarge] (JSON envelope message for code 2003),
//   - pkg/errors.ErrFileExceedsMaxUploadSize (errors.New / errors.Is sentinel).
//
// Do not copy this literal into messages.go or upload_errors.go — import constants.MsgFileTooLargeUpload.
const MsgFileTooLargeUpload = "uploaded file exceeds the maximum allowed size (2 GiB per file)"

// MsgMediaMultipartTotalTooLarge is used when the sum of file sizes in one multipart request exceeds MaxMediaMultipartTotalBytes.
const MsgMediaMultipartTotalTooLarge = "combined multipart files exceed the maximum allowed total size (2 GiB per request)"

// MsgMediaTooManyFilesInRequest is used when more than MaxMediaFilesPerRequest parts are submitted.
const MsgMediaTooManyFilesInRequest = "too many file parts in request (maximum 5)"

// MsgMediaFilesRequired is used when create/update expects at least one file part.
const MsgMediaFilesRequired = "at least one file part is required (multipart fields: files or file)"

// MsgMediaBatchDeleteTooManyIDs is used when batch delete lists more than MaxMediaBatchDelete keys.
const MsgMediaBatchDeleteTooManyIDs = "too many object keys in batch delete (maximum 10)"

// MsgMediaDuplicateObjectKeysInBatchDelete is used when batch delete lists duplicate keys.
const MsgMediaDuplicateObjectKeysInBatchDelete = "duplicate object keys in batch delete request"

// MsgBatchDeleteEmptyObjectKeys is used when batch delete is invoked with no keys (service-level guard).
const MsgBatchDeleteEmptyObjectKeys = "batch delete: empty object_keys"

// Default JSON messages for media upstream errcodes 9010–9014 (pkg/errcode/messages.go references only).
const (
	MsgMediaB2BucketNotConfigured    = "Media B2 bucket is not configured (set MEDIA_B2_BUCKET / media.b2_bucket so URLs can use path <cdn>/<bucket>/<object>)"
	MsgMediaBunnyStreamNotConfigured = "Bunny Stream is not configured (library id and API key required)"
	MsgMediaBunnyCreateFailed        = "Bunny Stream failed to create video"
	MsgMediaBunnyUploadFailed        = "Bunny Stream failed to upload video content"
	MsgMediaBunnyInvalidResponse     = "Bunny Stream returned an invalid response"
	MsgMediaBunnyVideoNotFound       = "Bunny Stream video was not found"
	MsgMediaBunnyGetVideoFailed      = "Bunny Stream failed to get video details"
	MsgMediaOptimisticLockConflict   = "media row was modified by another request; refresh and retry"
	MsgMediaReuseMismatch            = "reuse_media_id does not match this media row"

	// --- Sub 11 ---
	MsgExecutableUploadRejected = "file type is not allowed: executable and script files cannot be uploaded"
	MsgImageEncodeBusy          = "image encoder is at capacity; please retry (concurrent encode limit reached)"
)

// MsgInvalidProfileMediaFile is the sentinel text for pkg/errors.ErrInvalidProfileMediaFile
// (taxonomy category image_file_id, PATCH /me avatar_file_id — READY non-video raster image).
const MsgInvalidProfileMediaFile = "invalid profile or taxonomy image media file"

// --- pkg/errors sentinels (single source for message text; do not duplicate in errors.New) ---

// MsgNotFound is the sentinel text for pkg/errors.ErrNotFound (maps from gorm.ErrRecordNotFound).
const MsgNotFound = "not found"

// System privileged login / system_app_config (pkg/errors/system.go).
const (
	MsgSystemAppConfigMissing = "system_app_config row missing"
	MsgSystemSecretsNotReady  = "system secrets are not configured in database"
	MsgSystemLoginFailed      = "invalid system credentials"
)

// Auth user/session sentinels (pkg/errors/auth.go); wording kept aligned with prior services/auth strings.
const (
	MsgAuthEmailAlreadyExists  = "email already registered"
	MsgAuthInvalidCredentials  = "invalid email or password"
	MsgAuthWeakPassword        = "password does not meet requirements"
	MsgAuthEmailNotConfirmed   = "email not confirmed"
	MsgAuthUserDisabled        = "user account is disabled"
	MsgAuthInvalidConfirmToken = "invalid or expired confirmation token"
	MsgAuthUserNotFound        = "user not found"
	MsgAuthInvalidSession      = "invalid session"
	MsgAuthRefreshTokenExpired = "refresh token expired"
)

// MsgMediaDependencyNotConfigured is returned when media cloud clients are nil (RequireInitialized).
// JSON handlers still map this to errcode.InternalError + DefaultMessage(InternalError).
const MsgMediaDependencyNotConfigured = "internal server error"

// --- Repository / services nil DB guard (pkg/errors/common.go) ---

// MsgNilDatabase is returned when a function requires *gorm.DB but receives nil.
const MsgNilDatabase = "nil database"

// --- System privileged user registration (pkg/errors) ---

// MsgSystemUsernamePasswordRequired is returned when RegisterSystemPrivilegedUser receives empty credentials.
const MsgSystemUsernamePasswordRequired = "username and password required"

// --- RBAC (pkg/errors/rbac.go) ---

const (
	MsgRBACDatabaseNotConfigured         = "rbac database not configured"
	MsgRBACInvalidUserID                 = "invalid user id"
	MsgRBACPermissionIDRequired          = "permission_id required"
	MsgRBACUserAndPermissionNameRequired = "user id and permission_name required"
	MsgRBACRoleNameRequired              = "role name required"
	MsgRBACUnknownPermissionID           = "unknown permission_id"
	MsgRBACPermissionIDTooLong           = "permission_id too long (max 10)"
	MsgRBACPermissionNameRequired        = "permission_name required"
	MsgRBACPermissionNameTooLong         = "permission_name too long (max 50)"
)

// --- Media services / Bunny decode (pkg/errors) ---

const (
	MsgMediaVideoGUIDRequired              = "video guid is required"
	MsgMediaObjectKeyRequired              = "object key is required"
	MsgMediaFileNotFoundForObjectKey       = "media file not found for object_key"
	MsgBunnyStreamResponseMissingVideoGUID = "bunny stream did not return video guid"
	MsgBunnyGetVideoHTTP                   = "bunny get video: HTTP %d"
	MsgInvalidMetadataJSON                 = "invalid metadata json: %w"
	MsgGenerateSessionNonce                = "generate session nonce: %w"
	MsgJWTMissingSecretOrToken             = "missing secret or token"
	MsgJWTUnexpectedSigningMethod          = "unexpected signing method: %v"
	MsgJWTInvalidToken                     = "invalid token"
	MsgJWTRefreshMissingUserID             = "refresh token missing user_id"
	MsgSupabaseDBPoolNotInitialized        = "supabase database pool not initialized: set [supabase].DBURL"
	MsgSupabaseURLOrServiceKeyRequired     = "missing supabase URL or service role key"
	MsgRefreshSessionUnsupportedType       = "RefreshTokenSessionMap: unsupported source type %T"
	MsgSetupMediaClientsFailed             = "setup media clients failed: %w"
	MsgExpectedRowVersionInteger           = "expected_row_version must be an integer"
	MsgB2ClientNotConfigured               = "B2 client is not configured"
	MsgBunnyCreateVideoHTTP                = "bunny create video: HTTP %d"
	MsgBunnyUploadVideoHTTP                = "bunny upload video: HTTP %d"
	MsgBunnyStreamNotConfiguredRaw         = "bunny stream is not configured"
	MsgBunnyDeleteVideoFailed              = "bunny delete video failed: %s"
	MsgReadYAML                            = "read yaml %s: %w"
	MsgParseYAML                           = "parse yaml %s: %w"
	MsgParseDotEnv                         = "parse %s: %w"
	MsgRBACCatalogStructRequired           = "constants.AllPermissions must be a struct"
	MsgRandomDigitsPanicPrefix             = "GenerateRandomDigits: "
	MsgMigrationDBOpen                     = "migration db open: %w"
	MsgMigrationDBPing                     = "migration db ping: %w"
	MsgMigrationSource                     = "migration source: %w"
	MsgMigrationPostgresDriver             = "migration postgres driver: %w"
	MsgMigrationRun                        = "migrate: %w"
	MsgBrevoMarshalPayload                 = "brevo: marshal payload: %w"
	MsgBrevoBuildRequest                   = "brevo: build request: %w"
	MsgBrevoSendRequest                    = "brevo: send request: %w"
	MsgBrevoUnexpectedStatus               = "brevo: unexpected status %d"
	MsgBrevoRenderTemplate                 = "brevo: render template: %w"
)

// --- Signed local media URL token (pkg/errors/local_media_token.go) ---

const (
	MsgLocalMediaTokenInvalidFormat    = "invalid token format"
	MsgLocalMediaTokenInvalidPayload   = "invalid token payload"
	MsgLocalMediaTokenInvalidSignature = "invalid token signature"
	MsgLocalMediaTokenInvalid          = "invalid local media token"
)
