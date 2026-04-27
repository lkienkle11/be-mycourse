package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
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
	// backward-compat fallback for files uploaded before DB persistence existed.
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
		Metadata:  entities.DocumentMetadata{FileMetadata: entities.FileMetadata{}},
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
	meta := helper.NormalizeMetadata(req.Metadata)
	kind := helper.ResolveMediaKind(req.Kind, fileHeader.Header.Get("Content-Type"), filename)
	objectKey := helper.ResolveMediaUploadObjectKey(req.ObjectKey, filename, helper.DefaultMediaProvider(kind))
	provider := helper.DefaultMediaProvider(kind)

	if fileHeader.Size >= 0 && fileHeader.Size > constants.MaxMediaUploadFileBytes {
		return nil, pkgmedia.ErrFileExceedsMaxUploadSize
	}

	// Never read more than cap+1 bytes so oversized streams fail without buffering the whole payload.
	limited := io.LimitReader(file, constants.MaxMediaUploadFileBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > constants.MaxMediaUploadFileBytes {
		return nil, pkgmedia.ErrFileExceedsMaxUploadSize
	}

	uploaded, err := uploadToProvider(clients, provider, objectKey, filename, payload, meta)
	if err != nil {
		return nil, err
	}
	uploadedMetadata := entities.RawMetadata{}
	if metaMap, ok := uploaded.Metadata.(map[string]any); ok {
		uploadedMetadata = helper.NormalizeMetadata(metaMap)
	}
	sizeBytes := fileHeader.Size
	if sizeBytes < 0 {
		sizeBytes = int64(len(payload))
	}
	if sizeBytes == 0 {
		sizeBytes = int64(len(payload))
	}
	typedMetadata := helper.BuildTypedMetadata(
		kind,
		fileHeader.Header.Get("Content-Type"),
		filename,
		sizeBytes,
		payload,
		uploadedMetadata,
	)
	videoMeta, _ := typedMetadata.(entities.VideoMetadata)
	bunnyVideoID := strings.TrimSpace(fmt.Sprintf("%v", uploadedMetadata["bunny_video_id"]))
	if bunnyVideoID == "" {
		bunnyVideoID = strings.TrimSpace(fmt.Sprintf("%v", uploadedMetadata["video_guid"]))
	}
	bunnyLibraryID := strings.TrimSpace(fmt.Sprintf("%v", uploadedMetadata["bunny_library_id"]))
	videoProvider := strings.TrimSpace(fmt.Sprintf("%v", uploadedMetadata["video_provider"]))
	duration := int64(videoMeta.Duration)
	if duration <= 0 {
		duration = int64(utils.FloatFromRaw(uploadedMetadata, "length"))
	}
	now := time.Now()
	b2Bucket := strings.TrimSpace(setting.MediaSetting.B2Bucket)
	if b2Bucket == "" {
		b2Bucket = strings.TrimSpace(fmt.Sprintf("%v", uploadedMetadata["b2_bucket_name"]))
	}
	entity := &entities.File{
		ID:             uuid.NewString(),
		Kind:           kind,
		Provider:       provider,
		Filename:       filename,
		MimeType:       fileHeader.Header.Get("Content-Type"),
		SizeBytes:      sizeBytes,
		URL:            uploaded.URL,
		OriginURL:      uploaded.OriginURL,
		ObjectKey:      uploaded.ObjectKey,
		Status:         constants.FileStatusReady,
		B2BucketName:   b2Bucket,
		BunnyVideoID:   bunnyVideoID,
		BunnyLibraryID: bunnyLibraryID,
		Duration:       duration,
		VideoProvider:  videoProvider,
		Metadata:       typedMetadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	record := mapping.ToMediaModel(*entity)
	record.B2BucketName = strings.TrimSpace(setting.MediaSetting.B2Bucket)
	if err := mediaRepository().UpsertByObjectKey(record); err != nil {
		// Best-effort compensation: remove cloud object when DB write fails to avoid orphaned object.
		_ = DeleteFile(entity.ObjectKey, entities.RawMetadata{
			"video_guid": entity.BunnyVideoID,
		})
		return nil, err
	}
	return entity, nil
}

func UpdateFile(objectKey string, req dto.UpdateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	if strings.TrimSpace(objectKey) == "" {
		return nil, fmt.Errorf("object key is required")
	}
	prev, _ := mediaRepository().GetByObjectKey(strings.TrimSpace(objectKey))
	createReq := dto.CreateFileRequest{
		Kind:      req.Kind,
		ObjectKey: objectKey,
		Metadata:  req.Metadata,
	}
	row, err := CreateFile(createReq, file, fileHeader)
	if err != nil {
		return nil, err
	}
	row.ID = objectKey
	if prev != nil {
		row.CreatedAt = prev.CreatedAt
	}
	return row, nil
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
	if videoGUID := strings.TrimSpace(fmt.Sprintf("%v", metadata["video_guid"])); videoGUID != "" {
		provider = helper.DefaultMediaProvider(constants.FileKindVideo)
	}
	switch provider {
	case constants.FileProviderBunny:
		videoGUID := strings.TrimSpace(fmt.Sprintf("%v", metadata["video_guid"]))
		if videoGUID == "" {
			videoGUID = key
		}
		return clients.DeleteBunnyVideo(context.Background(), videoGUID)
	case constants.FileProviderLocal:
		return nil
	default:
		if err := clients.DeleteB2Object(context.Background(), key); err != nil {
			return err
		}
	}
	return mediaRepository().SoftDeleteByObjectKey(key)
}

func uploadToProvider(clients *pkgmedia.CloudClients, provider constants.FileProvider, objectKey, filename string, payload []byte, meta entities.RawMetadata) (dto.UploadFileResponse, error) {
	switch provider {
	case constants.FileProviderLocal:
		return clients.UploadLocal(objectKey, meta)
	case constants.FileProviderBunny:
		return clients.UploadBunnyVideo(context.Background(), filename, payload, objectKey, meta)
	default:
		return clients.UploadB2(context.Background(), objectKey, bytes.NewReader(payload), meta)
	}
}
