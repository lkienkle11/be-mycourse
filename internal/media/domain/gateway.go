package domain

import (
	"context"
	"mime/multipart"
)

// MediaGateway is the application-facing port for cloud storage, metadata, webhooks, and multipart IO.
type MediaGateway interface {
	RequireCloudReady() error
	DefaultMediaProvider(kind string) string
	BuildPublicURL(provider, objectKey string) (string, error)
	NormalizeMetadata(in map[string]any) RawMetadata
	ParseMetadataFromRaw(raw string) (RawMetadata, error)
	DecodeLocalURLToken(token string) (string, error)
	ResolveMediaKindFromServer(mime, filename string) (kind string, kindInferred bool)
	ResolveUploadProvider(kind string, kindInferred bool) string
	ResolveMediaUploadObjectKey(reqObjectKey, filename, provider string) string
	IsImageMIMEOrExt(mime, filename string) bool
	UploadToProvider(ctx context.Context, provider, objectKey, filename string, payload []byte, meta RawMetadata) (ProviderUploadResult, error)
	DeleteStoredObject(ctx context.Context, objectKey, provider, bunnyVideoID string) error
	BuildMediaFileEntityFromUpload(in MediaUploadEntityInput) *File
	GetBunnyVideoByID(ctx context.Context, videoGUID string) (*BunnyVideoDetail, error)
	BunnyStatusString(status int) string
	ProfileImageFileAcceptable(kind, mimeType, filename string) bool
	ShouldEnqueueSupersededCloudCleanup(prevObjectKey, prevBunnyVideoID, newObjectKey, newBunnyVideoID string) bool
	MergeMediaMetadataJSON(prevJSON []byte, overlay RawMetadata) ([]byte, error)
	ApplyBunnyDetailToMetadata(meta RawMetadata, d *BunnyVideoDetail, libraryID, streamPlayBase string)
	ApplyBunnyStreamFileColumns(f *File, d *BunnyVideoDetail, libraryID, streamPlayBase string)
	BuildTypedMetadata(kind, mimeType, filename string, sizeBytes int64, payload []byte, raw RawMetadata) UploadFileMetadata
	ApplyTypedMetadataToRaw(raw RawMetadata, typed UploadFileMetadata)
	EffectiveBunnyThumbnailURL(d *BunnyVideoDetail) string
	BunnyWebhookSigningSecret() string
	IsBunnyWebhookSignatureValid(rawBody []byte, signature, version, algorithm, secret string) bool
	CollectMultipartFileHeaders(form *multipart.Form) []*multipart.FileHeader
	ValidateMultipartFileHeaders(headers []*multipart.FileHeader) error
	OpenUploadParts(headers []*multipart.FileHeader) ([]OpenedUploadPart, error)
	CloseOpenedUploadParts(parts []OpenedUploadPart)
}
