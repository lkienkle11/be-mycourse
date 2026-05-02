package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/helper"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/repository"
	mediarepo "mycourse-io-be/repository/media"
)

func mediaRepository() *mediarepo.FileRepository {
	return repository.New(models.DB).Media
}

func enqueueSupersededCloudCleanup(repo *mediarepo.FileRepository, prevObjectKey string, prevProvider constants.FileProvider, prevBunnyVideoID string) {
	row := &models.MediaPendingCloudCleanup{
		Provider:     prevProvider,
		ObjectKey:    strings.TrimSpace(prevObjectKey),
		BunnyVideoID: strings.TrimSpace(prevBunnyVideoID),
	}
	_ = repo.InsertPendingCleanup(row)
}

func ListFiles(filter dto.FileFilter) ([]entities.File, int64, error) {
	rows, total, err := mediaRepository().List(filter)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entities.File, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapping.ToMediaEntity(row))
	}
	return out, total, nil
}

func GetFile(objectKey string, kind constants.FileKind) (*entities.File, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	row, err := mediaRepository().GetByObjectKey(key)
	if err == nil {
		entity := mapping.ToMediaEntity(*row)
		return &entity, nil
	}
	resolvedProvider := helper.DefaultMediaProvider(kind)
	fileURL := pkgmedia.BuildPublicURL(resolvedProvider, key)
	now := time.Now()
	return &entities.File{
		ID:        key,
		Kind:      kind,
		Provider:  resolvedProvider,
		Filename:  key,
		MimeType:  "",
		SizeBytes: 0,
		URL:       fileURL,
		OriginURL: fileURL,
		ObjectKey: key,
		Status:    constants.FileStatusReady,
		Metadata:  entities.UploadFileMetadata{},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func CreateFile(req dto.CreateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	clients := pkgmedia.Cloud

	filename := strings.TrimSpace(fileHeader.Filename)
	kind, kindInferred := helper.ResolveMediaKindFromServer(fileHeader.Header.Get("Content-Type"), filename)
	provider := helper.ResolveUploadProvider(kind, kindInferred)
	objectKey := helper.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)

	if fileHeader.Size >= 0 && fileHeader.Size > constants.MaxMediaUploadFileBytes {
		return nil, pkgerrors.ErrFileExceedsMaxUploadSize
	}

	limited := io.LimitReader(file, constants.MaxMediaUploadFileBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > constants.MaxMediaUploadFileBytes {
		return nil, pkgerrors.ErrFileExceedsMaxUploadSize
	}

	mime := fileHeader.Header.Get("Content-Type")
	isImage := helper.IsImageMIMEOrExt(mime, filename)

	// Reject executable/script files (extension + magic bytes) for non-image, non-video uploads.
	if kind == constants.FileKindFile && !isImage {
		head := payload
		if len(head) > 16 {
			head = head[:16]
		}
		if utils.IsExecutableUploadRejected(filename, head) {
			return nil, pkgerrors.ErrExecutableUploadRejected
		}
	}

	// Convert image uploads to WebP before sending to the storage provider.
	if isImage {
		utils.AcquireEncodeGate()
		encoded, newMime, encErr := utils.EncodeWebP(payload)
		utils.ReleaseEncodeGate()
		if encErr != nil {
			return nil, &pkgerrors.ProviderError{
				Code: errcode.ImageEncodeBusy,
				Msg:  encErr.Error(),
				Err:  encErr,
			}
		}
		payload = encoded
		mime = newMime
		if ext := filepath.Ext(filename); ext != "" {
			filename = strings.TrimSuffix(filename, ext) + ".webp"
		}
	}

	uploaded, err := uploadToProvider(clients, provider, objectKey, filename, payload, entities.RawMetadata{})
	if err != nil {
		return nil, err
	}

	uploadedMeta := helper.NormalizeMetadata(uploaded.Metadata)
	merged := helper.NormalizeMetadata(uploadedMeta)

	sizeBytes := fileHeader.Size
	if sizeBytes < 0 || sizeBytes == 0 {
		sizeBytes = int64(len(payload))
	}
	if isImage {
		// After WebP conversion payload length differs from original; use actual encoded size.
		sizeBytes = int64(len(payload))
	}

	now := time.Now()
	input := entities.MediaUploadEntityInput{
		Kind:          kind,
		Provider:      provider,
		Filename:      filename,
		ContentType:   mime,
		SizeBytes:     sizeBytes,
		Payload:       payload,
		Uploaded:      uploaded,
		UploadedMeta:  merged,
		B2Bucket:      strings.TrimSpace(setting.MediaSetting.B2Bucket),
		CreatedAt:     now,
		UpdatedAt:     now,
		GenerateNewID: true,
		PreserveID:    "",
	}

	entity := helper.BuildMediaFileEntityFromUpload(input)
	record := mapping.ToMediaModel(*entity)
	record.B2BucketName = strings.TrimSpace(setting.MediaSetting.B2Bucket)
	record.ContentFingerprint = utils.ContentFingerprint(payload)

	repo := mediaRepository()
	if err := repo.UpsertByObjectKey(record); err != nil {
		_ = pkgmedia.DeleteStoredObject(context.Background(), clients, entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}
	out, err := repo.GetByObjectKey(entity.ObjectKey)
	if err != nil {
		fallback := mapping.ToMediaEntity(*record)
		return &fallback, nil
	}
	ent := mapping.ToMediaEntity(*out)
	return &ent, nil
}

func UpdateFile(objectKey string, req dto.UpdateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	repo := mediaRepository()
	prevRow, err := repo.GetByObjectKey(key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("media file not found for object_key")
		}
		return nil, err
	}

	if rid := strings.TrimSpace(req.ReuseMediaID); rid != "" && rid != prevRow.ID {
		return nil, pkgerrors.ErrMediaReuseMismatch
	}
	if req.ExpectedRowVersion != nil && *req.ExpectedRowVersion != prevRow.RowVersion {
		return nil, pkgerrors.ErrMediaOptimisticLock
	}

	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	clients := pkgmedia.Cloud

	prevRaw := entities.RawMetadata{}
	_ = json.Unmarshal(prevRow.MetadataJSON, &prevRaw)

	filename := strings.TrimSpace(fileHeader.Filename)
	if fileHeader.Size >= 0 && fileHeader.Size > constants.MaxMediaUploadFileBytes {
		return nil, pkgerrors.ErrFileExceedsMaxUploadSize
	}

	limited := io.LimitReader(file, constants.MaxMediaUploadFileBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > constants.MaxMediaUploadFileBytes {
		return nil, pkgerrors.ErrFileExceedsMaxUploadSize
	}

	mime := fileHeader.Header.Get("Content-Type")
	isImage := helper.IsImageMIMEOrExt(mime, filename)

	fp := utils.ContentFingerprint(payload)
	if req.SkipUploadIfUnchanged && prevRow.ContentFingerprint != "" && fp == prevRow.ContentFingerprint {
		blob, err := helper.MergeMediaMetadataJSON(prevRow.MetadataJSON, entities.RawMetadata{})
		if err != nil {
			return nil, err
		}
		rec := *prevRow
		rec.MetadataJSON = blob
		rec.UpdatedAt = time.Now()
		if filename != "" {
			rec.Filename = filename
		}
		if err := repo.SaveWithRowVersionCheck(&rec, prevRow.RowVersion); err != nil {
			return nil, err
		}
		saved, err := repo.GetByID(prevRow.ID)
		if err != nil {
			return nil, err
		}
		ent := mapping.ToMediaEntity(*saved)
		return &ent, nil
	}

	kind, kindInferred := helper.ResolveMediaKindFromServer(fileHeader.Header.Get("Content-Type"), filename)
	provider := helper.ResolveUploadProvider(kind, kindInferred)
	resolvedObjectKey := helper.ResolveMediaUploadObjectKey("", filename, provider)

	// Convert image uploads to WebP before sending to the storage provider.
	if isImage {
		utils.AcquireEncodeGate()
		encoded, newMime, encErr := utils.EncodeWebP(payload)
		utils.ReleaseEncodeGate()
		if encErr != nil {
			return nil, &pkgerrors.ProviderError{
				Code: errcode.ImageEncodeBusy,
				Msg:  encErr.Error(),
				Err:  encErr,
			}
		}
		payload = encoded
		mime = newMime
		if ext := filepath.Ext(filename); ext != "" {
			filename = strings.TrimSuffix(filename, ext) + ".webp"
		}
	}

	uploaded, err := uploadToProvider(clients, provider, resolvedObjectKey, filename, payload, entities.RawMetadata{})
	if err != nil {
		return nil, err
	}

	uploadedMeta := helper.NormalizeMetadata(uploaded.Metadata)
	merged := helper.NormalizeMetadata(uploadedMeta)
	for k, v := range prevRaw {
		if _, ok := merged[k]; !ok {
			merged[k] = v
		}
	}

	sizeBytes := fileHeader.Size
	if sizeBytes < 0 || sizeBytes == 0 {
		sizeBytes = int64(len(payload))
	}
	if isImage {
		sizeBytes = int64(len(payload))
	}

	now := time.Now()
	input := entities.MediaUploadEntityInput{
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

	entity := helper.BuildMediaFileEntityFromUpload(input)
	record := mapping.ToMediaModel(*entity)
	record.B2BucketName = strings.TrimSpace(setting.MediaSetting.B2Bucket)
	record.ContentFingerprint = fp

	if err := repo.SaveWithRowVersionCheck(record, prevRow.RowVersion); err != nil {
		_ = pkgmedia.DeleteStoredObject(context.Background(), clients, entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}

	if helper.ShouldEnqueueSupersededCloudCleanup(prevRow.ObjectKey, prevRow.BunnyVideoID, entity.ObjectKey, entity.BunnyVideoID) {
		enqueueSupersededCloudCleanup(repo, prevRow.ObjectKey, prevRow.Provider, prevRow.BunnyVideoID)
	}

	saved, err := repo.GetByID(prevRow.ID)
	if err != nil {
		fallback := mapping.ToMediaEntity(*record)
		return &fallback, nil
	}
	ent := mapping.ToMediaEntity(*saved)
	return &ent, nil
}

func DeleteFile(objectKey string, metadata entities.RawMetadata) error {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
		return err
	}
	clients := pkgmedia.Cloud
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	provider := helper.DefaultMediaProvider(constants.FileKindFile)
	bunnyID := strings.TrimSpace(fmt.Sprintf("%v", metadata[constants.MediaMetaKeyVideoGUID]))
	if bunnyID == "" {
		bunnyID = strings.TrimSpace(fmt.Sprintf("%v", metadata[constants.MediaMetaKeyBunnyVideoID]))
	}
	if bunnyID != "" {
		provider = helper.DefaultMediaProvider(constants.FileKindVideo)
	}
	if err := pkgmedia.DeleteStoredObject(context.Background(), clients, key, provider, bunnyID); err != nil {
		return err
	}
	return mediaRepository().SoftDeleteByObjectKey(key)
}

func uploadToProvider(clients *entities.CloudClients, provider constants.FileProvider, objectKey, filename string, payload []byte, meta entities.RawMetadata) (entities.ProviderUploadResult, error) {
	switch provider {
	case constants.FileProviderLocal:
		return pkgmedia.UploadLocal(clients, objectKey, meta)
	case constants.FileProviderBunny:
		return pkgmedia.UploadBunnyVideo(clients, context.Background(), filename, payload, objectKey, meta)
	default:
		return pkgmedia.UploadB2(clients, context.Background(), objectKey, bytes.NewReader(payload), meta)
	}
}
