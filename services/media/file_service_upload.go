package media

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"strings"
	"time"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	mediarepo "mycourse-io-be/repository/media"
)

// normalizeUpdateMultipartPayload resolves kind/provider/object key, rejects unsafe payloads,
// and optionally re-encodes images to WebP for an update body.
func normalizeUpdateMultipartPayload(filename, mime string, payload []byte) (
	newPayload []byte, newFilename, newMime string, kind string, provider string, objectKey string, err error,
) {
	kind, kindInferred := pkgmedia.ResolveMediaKindFromServer(mime, filename)
	provider = pkgmedia.ResolveUploadProvider(kind, kindInferred)
	objectKey = pkgmedia.ResolveMediaUploadObjectKey("", filename, provider)
	isImage := pkgmedia.IsImageMIMEOrExt(mime, filename)
	if err = rejectExecutableNonMedia(kind, isImage, filename, payload); err != nil {
		return
	}
	if isImage {
		var enc []byte
		var encMime, encName string
		var encErr error
		enc, encMime, encName, encErr = encodeUploadToWebP(payload, filename)
		if encErr != nil {
			err = encErr
			return
		}
		newPayload, newMime, newFilename = enc, encMime, encName
		objectKey = pkgmedia.ResolveMediaUploadObjectKey("", newFilename, provider)
		return
	}
	newPayload, newMime, newFilename = payload, mime, filename
	return
}

func mediaUploadEntityInputForRowUpdate(
	prevRow *models.MediaFile,
	kind string,
	provider string,
	filename, mime string,
	sizeBytes int64,
	payload []byte,
	uploaded entities.ProviderUploadResult,
	merged entities.RawMetadata,
) entities.MediaUploadEntityInput {
	now := time.Now()
	return entities.MediaUploadEntityInput{
		Kind:          kind,
		Provider:      provider,
		Filename:      filename,
		ContentType:   mime,
		SizeBytes:     sizeBytes,
		Payload:       payload,
		Uploaded:      uploaded,
		UploadedMeta:  merged,
		B2Bucket:      strings.TrimSpace(setting.MediaSetting.B2Bucket),
		CreatedAt:     prevRow.CreatedAt,
		UpdatedAt:     now,
		GenerateNewID: false,
		PreserveID:    prevRow.ID,
	}
}

func runUpdateFileMultipartBody(repo *mediarepo.FileRepository, clients *entities.CloudClients, prevRow *models.MediaFile, req dto.UpdateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	prevRaw := entities.RawMetadata{}
	_ = json.Unmarshal(prevRow.MetadataJSON, &prevRaw)

	payload, filename, mime, err := readMultipartPayloadLimited(file, fileHeader)
	if err != nil {
		return nil, err
	}
	fp := utils.ContentFingerprint(payload)
	if req.SkipUploadIfUnchanged && prevRow.ContentFingerprint != "" && fp == prevRow.ContentFingerprint {
		return saveUnchangedFingerprintMetadata(repo, prevRow, filename, prevRow.RowVersion)
	}

	payload, filename, mime, kind, provider, resolvedObjectKey, err := normalizeUpdateMultipartPayload(filename, mime, payload)
	if err != nil {
		return nil, err
	}
	isImage := pkgmedia.IsImageMIMEOrExt(mime, filename)

	uploaded, err := uploadToProvider(clients, provider, resolvedObjectKey, filename, payload, entities.RawMetadata{})
	if err != nil {
		return nil, err
	}

	merged := mergeProviderMetadataWithPrevious(uploaded, prevRaw)
	sizeBytes := effectiveUploadSizeBytes(fileHeader.Size, payload, isImage)
	input := mediaUploadEntityInputForRowUpdate(prevRow, kind, provider, filename, mime, sizeBytes, payload, uploaded, merged)
	return persistUpdatedMediaRow(clients, repo, prevRow, input, payload, fp)
}

func uploadToProvider(clients *entities.CloudClients, provider string, objectKey, filename string, payload []byte, meta entities.RawMetadata) (entities.ProviderUploadResult, error) {
	switch provider {
	case constants.FileProviderLocal:
		return pkgmedia.UploadLocal(clients, objectKey, meta)
	case constants.FileProviderBunny:
		return pkgmedia.UploadBunnyVideo(clients, context.Background(), filename, payload, objectKey, meta)
	default:
		return pkgmedia.UploadB2(clients, context.Background(), objectKey, bytes.NewReader(payload), meta)
	}
}
