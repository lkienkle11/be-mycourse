package infra

import (
	"bytes"
	"context"
	"mime/multipart"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
)

// StorageGateway implements domain.MediaGateway using package-level infra helpers.
type StorageGateway struct{}

// NewStorageGateway constructs the default MediaGateway adapter.
func NewStorageGateway() domain.MediaGateway {
	return StorageGateway{}
}

func (StorageGateway) RequireCloudReady() error {
	return RequireInitialized(Cloud)
}

func (StorageGateway) DefaultMediaProvider(kind string) string {
	return DefaultMediaProvider(kind)
}

func (StorageGateway) BuildPublicURL(provider, objectKey string) string {
	return BuildPublicURL(provider, objectKey)
}

func (StorageGateway) NormalizeMetadata(in map[string]any) domain.RawMetadata {
	return NormalizeMetadata(in)
}

func (StorageGateway) ParseMetadataFromRaw(raw string) (domain.RawMetadata, error) {
	return ParseMetadataFromRaw(raw)
}

func (StorageGateway) DecodeLocalURLToken(token string) (string, error) {
	return DecodeLocalURLToken(token)
}

func (StorageGateway) ResolveMediaKindFromServer(mime, filename string) (string, bool) {
	return ResolveMediaKindFromServer(mime, filename)
}

func (StorageGateway) ResolveUploadProvider(kind string, kindInferred bool) string {
	return ResolveUploadProvider(kind, kindInferred)
}

func (StorageGateway) ResolveMediaUploadObjectKey(reqObjectKey, filename, provider string) string {
	return ResolveMediaUploadObjectKey(reqObjectKey, filename, provider)
}

func (StorageGateway) IsImageMIMEOrExt(mime, filename string) bool {
	return IsImageMIMEOrExt(mime, filename)
}

func (StorageGateway) UploadToProvider(ctx context.Context, provider, objectKey, filename string, payload []byte, meta domain.RawMetadata) (domain.ProviderUploadResult, error) {
	switch provider {
	case constants.FileProviderLocal:
		return UploadLocal(Cloud, objectKey, meta)
	case constants.FileProviderBunny:
		return UploadBunnyVideo(Cloud, ctx, filename, payload, objectKey, meta)
	default:
		return UploadB2(Cloud, ctx, objectKey, bytes.NewReader(payload), meta)
	}
}

func (StorageGateway) DeleteStoredObject(ctx context.Context, objectKey, provider, bunnyVideoID string) error {
	return DeleteStoredObject(ctx, Cloud, objectKey, provider, bunnyVideoID)
}

func (StorageGateway) BuildMediaFileEntityFromUpload(in domain.MediaUploadEntityInput) *domain.File {
	return BuildMediaFileEntityFromUpload(in)
}

func (StorageGateway) GetBunnyVideoByID(ctx context.Context, videoGUID string) (*domain.BunnyVideoDetail, error) {
	return GetBunnyVideoByID(Cloud, ctx, videoGUID)
}

func (StorageGateway) BunnyStatusString(status int) string {
	return BunnyStatusString(status)
}

func (StorageGateway) ProfileImageFileAcceptable(kind, mimeType, filename string) bool {
	return ProfileImageFileAcceptable(kind, mimeType, filename)
}

func (StorageGateway) ShouldEnqueueSupersededCloudCleanup(prevObjectKey, prevBunnyVideoID, newObjectKey, newBunnyVideoID string) bool {
	return ShouldEnqueueSupersededCloudCleanup(prevObjectKey, prevBunnyVideoID, newObjectKey, newBunnyVideoID)
}

func (StorageGateway) MergeMediaMetadataJSON(prevJSON []byte, overlay domain.RawMetadata) ([]byte, error) {
	return MergeMediaMetadataJSON(prevJSON, overlay)
}

func (StorageGateway) ApplyBunnyDetailToMetadata(meta domain.RawMetadata, d *domain.BunnyVideoDetail, libraryID, streamPlayBase string) {
	ApplyBunnyDetailToMetadata(meta, d, libraryID, streamPlayBase)
}

func (StorageGateway) ApplyBunnyStreamFileColumns(f *domain.File, d *domain.BunnyVideoDetail, libraryID, streamPlayBase string) {
	ApplyBunnyStreamFileColumns(f, d, libraryID, streamPlayBase)
}

func (StorageGateway) BuildTypedMetadata(kind, mimeType, filename string, sizeBytes int64, payload []byte, raw domain.RawMetadata) domain.UploadFileMetadata {
	return BuildTypedMetadata(kind, mimeType, filename, sizeBytes, payload, raw)
}

func (StorageGateway) ApplyTypedMetadataToRaw(raw domain.RawMetadata, typed domain.UploadFileMetadata) {
	ApplyTypedMetadataToRaw(raw, typed)
}

func (StorageGateway) EffectiveBunnyThumbnailURL(d *domain.BunnyVideoDetail) string {
	return EffectiveBunnyThumbnailURL(d)
}

func (StorageGateway) BunnyWebhookSigningSecret() string {
	return BunnyWebhookSigningSecret()
}

func (StorageGateway) IsBunnyWebhookSignatureValid(rawBody []byte, signature, version, algorithm, secret string) bool {
	return IsBunnyWebhookSignatureValid(rawBody, signature, version, algorithm, secret)
}

func (StorageGateway) CollectMultipartFileHeaders(form *multipart.Form) []*multipart.FileHeader {
	return CollectMultipartFileHeaders(form)
}

func (StorageGateway) ValidateMultipartFileHeaders(headers []*multipart.FileHeader) error {
	return ValidateMultipartFileHeaders(headers)
}

func (StorageGateway) OpenUploadParts(headers []*multipart.FileHeader) ([]domain.OpenedUploadPart, error) {
	return OpenUploadParts(headers)
}

func (StorageGateway) CloseOpenedUploadParts(parts []domain.OpenedUploadPart) {
	CloseOpenedUploadParts(parts)
}
