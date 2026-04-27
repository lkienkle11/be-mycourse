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
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/helper"
	pkgmedia "mycourse-io-be/pkg/media"
)

func ListFiles(_ dto.FileFilter) ([]entities.File, int64, error) {
	return []entities.File{}, 0, nil
}

func GetFile(objectKey string, kind constants.FileKind) (*entities.File, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	resolvedProvider := helper.DefaultMediaProvider(kind)
	fileURL := pkgmedia.BuildPublicURL(resolvedProvider, key)
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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
	typedMetadata := helper.BuildTypedMetadata(
		kind,
		fileHeader.Header.Get("Content-Type"),
		filename,
		sizeBytes,
		payload,
		uploadedMetadata,
	)
	now := time.Now()
	return &entities.File{
		ID:        uuid.NewString(),
		Kind:      kind,
		Provider:  provider,
		Filename:  filename,
		MimeType:  fileHeader.Header.Get("Content-Type"),
		SizeBytes: sizeBytes,
		URL:       uploaded.URL,
		OriginURL: uploaded.OriginURL,
		ObjectKey: uploaded.ObjectKey,
		Status:    constants.FileStatusReady,
		Metadata:  typedMetadata,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func UpdateFile(objectKey string, req dto.UpdateFileRequest, file multipart.File, fileHeader *multipart.FileHeader) (*entities.File, error) {
	if strings.TrimSpace(objectKey) == "" {
		return nil, fmt.Errorf("object key is required")
	}
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
		return clients.DeleteB2Object(context.Background(), key)
	}
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
