package media

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
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

func mergeProviderMetadataWithPrevious(upload entities.ProviderUploadResult, prev entities.RawMetadata) entities.RawMetadata {
	uploadedMeta := pkgmedia.NormalizeMetadata(upload.Metadata)
	merged := pkgmedia.NormalizeMetadata(uploadedMeta)
	for k, v := range prev {
		if _, ok := merged[k]; !ok {
			merged[k] = v
		}
	}
	return merged
}

func persistCreateMediaRow(clients *entities.CloudClients, input entities.MediaUploadEntityInput, payload []byte) (*entities.File, error) {
	entity := pkgmedia.BuildMediaFileEntityFromUpload(input)
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

func persistUpdatedMediaRow(clients *entities.CloudClients, repo *mediarepo.FileRepository, prevRow *models.MediaFile, input entities.MediaUploadEntityInput, payload []byte, fp string) (*entities.File, error) {
	entity := pkgmedia.BuildMediaFileEntityFromUpload(input)
	record := mapping.ToMediaModel(*entity)
	record.B2BucketName = strings.TrimSpace(setting.MediaSetting.B2Bucket)
	record.ContentFingerprint = fp

	if err := repo.SaveWithRowVersionCheck(record, prevRow.RowVersion); err != nil {
		_ = pkgmedia.DeleteStoredObject(context.Background(), clients, entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}
	if pkgmedia.ShouldEnqueueSupersededCloudCleanup(prevRow.ObjectKey, prevRow.BunnyVideoID, entity.ObjectKey, entity.BunnyVideoID) {
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

func saveUnchangedFingerprintMetadata(repo *mediarepo.FileRepository, prevRow *models.MediaFile, filename string, rowVersion int64) (*entities.File, error) {
	blob, err := pkgmedia.MergeMediaMetadataJSON(prevRow.MetadataJSON, entities.RawMetadata{})
	if err != nil {
		return nil, err
	}
	rec := *prevRow
	rec.MetadataJSON = blob
	rec.UpdatedAt = time.Now()
	if filename != "" {
		rec.Filename = filename
	}
	if err := repo.SaveWithRowVersionCheck(&rec, rowVersion); err != nil {
		return nil, err
	}
	saved, err := repo.GetByID(prevRow.ID)
	if err != nil {
		return nil, err
	}
	ent := mapping.ToMediaEntity(*saved)
	return &ent, nil
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
	resolvedProvider := pkgmedia.DefaultMediaProvider(kind)
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

// prepareCreateMultipartBody reads the upload, infers kind/provider/object key, rejects unsafe payloads,
// and optionally re-encodes images to WebP (updating filename/mime/objectKey accordingly).
func prepareCreateMultipartBody(req dto.CreateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (
	payload []byte, filename, mime string, kind constants.FileKind, provider constants.FileProvider, objectKey string, err error,
) {
	payload, filename, mime, err = readMultipartPayloadLimited(file, fileHeader)
	if err != nil {
		return
	}
	kind, kindInferred := pkgmedia.ResolveMediaKindFromServer(mime, filename)
	provider = pkgmedia.ResolveUploadProvider(kind, kindInferred)
	objectKey = pkgmedia.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)
	isImage := pkgmedia.IsImageMIMEOrExt(mime, filename)
	if err = rejectExecutableNonMedia(kind, isImage, filename, payload); err != nil {
		return
	}
	if isImage {
		var enc []byte
		var newMime, newName string
		var encErr error
		enc, newMime, newName, encErr = encodeUploadToWebP(payload, filename)
		if encErr != nil {
			err = encErr
			return
		}
		payload, mime, filename = enc, newMime, newName
		objectKey = pkgmedia.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)
	}
	return
}

func createFileEntityInput(
	fileHeader *multipart.FileHeader,
	payload []byte,
	filename, mime string,
	kind constants.FileKind,
	provider constants.FileProvider,
	uploaded entities.ProviderUploadResult,
	now time.Time,
) entities.MediaUploadEntityInput {
	uploadedMeta := pkgmedia.NormalizeMetadata(uploaded.Metadata)
	merged := pkgmedia.NormalizeMetadata(uploadedMeta)
	isImage := pkgmedia.IsImageMIMEOrExt(mime, filename)
	return entities.MediaUploadEntityInput{
		Kind:          kind,
		Provider:      provider,
		Filename:      filename,
		ContentType:   mime,
		SizeBytes:     effectiveUploadSizeBytes(fileHeader.Size, payload, isImage),
		Payload:       payload,
		Uploaded:      uploaded,
		UploadedMeta:  merged,
		B2Bucket:      strings.TrimSpace(setting.MediaSetting.B2Bucket),
		CreatedAt:     now,
		UpdatedAt:     now,
		GenerateNewID: true,
		PreserveID:    "",
	}
}

func CreateFile(req dto.CreateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	clients := pkgmedia.Cloud
	payload, filename, mime, kind, provider, objectKey, err := prepareCreateMultipartBody(req, file, fileHeader)
	if err != nil {
		return nil, err
	}
	uploaded, err := uploadToProvider(clients, provider, objectKey, filename, payload, entities.RawMetadata{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	input := createFileEntityInput(fileHeader, payload, filename, mime, kind, provider, uploaded, now)
	return persistCreateMediaRow(clients, input, payload)
}

func loadUpdateFileTarget(objectKey string, req dto.UpdateFileRequest) (*mediarepo.FileRepository, *models.MediaFile, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, nil, fmt.Errorf("object key is required")
	}
	repo := mediaRepository()
	prevRow, err := repo.GetByObjectKey(key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("media file not found for object_key")
		}
		return nil, nil, err
	}
	if rid := strings.TrimSpace(req.ReuseMediaID); rid != "" && rid != prevRow.ID {
		return nil, nil, pkgerrors.ErrMediaReuseMismatch
	}
	if req.ExpectedRowVersion != nil && *req.ExpectedRowVersion != prevRow.RowVersion {
		return nil, nil, pkgerrors.ErrMediaOptimisticLock
	}
	return repo, prevRow, nil
}

func UpdateFile(objectKey string, req dto.UpdateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	repo, prevRow, err := loadUpdateFileTarget(objectKey, req)
	if err != nil {
		return nil, err
	}

	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	clients := pkgmedia.Cloud

	return runUpdateFileMultipartBody(repo, clients, prevRow, req, file, fileHeader)
}

func DeleteFile(objectKey string, metadata entities.RawMetadata) error {
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return err
	}
	clients := pkgmedia.Cloud
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	provider := pkgmedia.DefaultMediaProvider(constants.FileKindFile)
	bunnyID := strings.TrimSpace(fmt.Sprintf("%v", metadata[constants.MediaMetaKeyVideoGUID]))
	if bunnyID == "" {
		bunnyID = strings.TrimSpace(fmt.Sprintf("%v", metadata[constants.MediaMetaKeyBunnyVideoID]))
	}
	if bunnyID != "" {
		provider = pkgmedia.DefaultMediaProvider(constants.FileKindVideo)
	}
	if err := pkgmedia.DeleteStoredObject(context.Background(), clients, key, provider, bunnyID); err != nil {
		return err
	}
	return mediaRepository().SoftDeleteByObjectKey(key)
}
