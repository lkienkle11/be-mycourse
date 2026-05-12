package domain

import stderrors "errors"

// Media domain sentinel errors.
var (
	ErrMediaOptimisticLock            = stderrors.New("media row was modified by another request; refresh and retry")
	ErrMediaReuseMismatch             = stderrors.New("reuse_media_id does not match this media row")
	ErrExecutableUploadRejected       = stderrors.New("file type is not allowed: executable and script files cannot be uploaded")
	ErrImageEncodeBusy                = stderrors.New("image encoder is at capacity; please retry (concurrent encode limit reached)")
	ErrDependencyNotConfigured        = stderrors.New("internal server error")
	ErrMediaVideoGUIDRequired         = stderrors.New("video guid is required")
	ErrMediaObjectKeyRequired         = stderrors.New("object key is required")
	ErrMediaFileNotFoundForObjectKey  = stderrors.New("media file not found for object_key")
	ErrBunnyStreamResponseMissingGUID = stderrors.New("bunny stream did not return video guid")

	ErrFileExceedsMaxUploadSize           = stderrors.New("uploaded file exceeds the maximum allowed size (2 GiB per file)")
	ErrMediaMultipartTotalTooLarge        = stderrors.New("combined multipart files exceed the maximum allowed total size (2 GiB per request)")
	ErrMediaTooManyFilesInRequest         = stderrors.New("too many file parts in request (maximum 5)")
	ErrMediaFilesRequired                 = stderrors.New("at least one file part is required (multipart fields: files or file)")
	ErrMediaBatchDeleteTooManyIDs         = stderrors.New("too many object keys in batch delete (maximum 10)")
	ErrMediaDuplicateObjectKeysInBatch    = stderrors.New("duplicate object keys in batch delete request")
	ErrBatchDeleteEmptyKeys               = stderrors.New("batch delete: empty object_keys")

	ErrBunnyWebhookJSONInvalid    = stderrors.New("bunny webhook: invalid json")
	ErrBunnyWebhookPayloadInvalid = stderrors.New("bunny webhook: invalid payload")

	ErrInvalidProfileMediaFile = stderrors.New("invalid profile or taxonomy image media file")

	ErrLocalMediaTokenInvalidFormat    = stderrors.New("invalid token format")
	ErrLocalMediaTokenInvalidPayload   = stderrors.New("invalid token payload")
	ErrLocalMediaTokenInvalidSignature = stderrors.New("invalid token signature")
	ErrLocalMediaTokenInvalid          = stderrors.New("invalid local media token")
)

// ProviderError carries an application errcode for B2/Bunny client failures.
type ProviderError struct {
	Code int
	Msg  string
	Err  error
}

func (e *ProviderError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return e.Msg
}

func (e *ProviderError) Unwrap() error { return e.Err }
