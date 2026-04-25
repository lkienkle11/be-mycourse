package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
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

func GetFile(objectKey string, provider constants.FileProvider, kind constants.FileKind) (*entities.File, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	resolvedProvider := provider
	if resolvedProvider == "" {
		if kind == constants.FileKindVideo {
			resolvedProvider = constants.FileProviderBunny
		} else {
			resolvedProvider = constants.FileProviderB2
		}
	}
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
		Metadata:  entities.FileMetadata{},
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
	kind := resolveKind(req.Kind, fileHeader.Header.Get("Content-Type"), filename)
	objectKey := pkgmedia.BuildObjectKey(req.ObjectKey, filename)
	provider := resolveProvider(kind, req.Provider)

	uploaded, err := uploadToProvider(clients, provider, objectKey, filename, file, meta)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &entities.File{
		ID:        uuid.NewString(),
		Kind:      kind,
		Provider:  provider,
		Filename:  filename,
		MimeType:  fileHeader.Header.Get("Content-Type"),
		SizeBytes: fileHeader.Size,
		URL:       uploaded.URL,
		OriginURL: uploaded.OriginURL,
		ObjectKey: uploaded.ObjectKey,
		Status:    constants.FileStatusReady,
		Metadata:  uploaded.Metadata,
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
		Provider:  req.Provider,
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

func DeleteFile(objectKey string, provider constants.FileProvider, metadata entities.FileMetadata) error {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
		return err
	}
	clients := pkgmedia.Cloud
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	switch provider {
	case constants.FileProviderBunny:
		videoGUID := strings.TrimSpace(fmt.Sprintf("%v", metadata["video_guid"]))
		if videoGUID == "" {
			videoGUID = key
		}
		return clients.DeleteBunnyVideo(contextBackground(), videoGUID)
	case constants.FileProviderLocal:
		return nil
	default:
		return clients.DeleteB2Object(contextBackground(), key)
	}
}

func DecodeLocalURLToken(token string) (string, error) {
	secret := strings.TrimSpace(os.Getenv("LOCAL_FILE_URL_SECRET"))
	if secret == "" {
		secret = "mycourse-local-file-secret"
	}
	objectKey, err := helper.DecodeLocalObjectKey(secret, token)
	if err != nil {
		return "", errors.New("invalid local media token")
	}
	return objectKey, nil
}

func resolveKind(kindRaw, mime, filename string) constants.FileKind {
	kind := constants.FileKind(strings.TrimSpace(kindRaw))
	if kind == constants.FileKindFile || kind == constants.FileKindVideo {
		return kind
	}
	if strings.HasPrefix(strings.ToLower(mime), "video/") {
		return constants.FileKindVideo
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".mov", ".mkv", ".avi", ".webm":
		return constants.FileKindVideo
	default:
		return constants.FileKindFile
	}
}

func resolveProvider(kind constants.FileKind, providerRaw string) constants.FileProvider {
	p := constants.FileProvider(strings.TrimSpace(providerRaw))
	if p != "" {
		return p
	}
	if kind == constants.FileKindVideo {
		return constants.FileProviderBunny
	}
	return constants.FileProviderB2
}

func uploadToProvider(clients *pkgmedia.CloudClients, provider constants.FileProvider, objectKey, filename string, file multipart.File, meta entities.FileMetadata) (dto.UploadFileResponse, error) {
	switch provider {
	case constants.FileProviderLocal:
		return clients.UploadLocal(objectKey, meta)
	case constants.FileProviderBunny:
		payload, err := io.ReadAll(file)
		if err != nil {
			return dto.UploadFileResponse{}, err
		}
		return clients.UploadBunnyVideo(contextBackground(), filename, payload, objectKey, meta)
	default:
		return clients.UploadB2(contextBackground(), objectKey, file, meta)
	}
}

func contextBackground() context.Context {
	return context.Background()
}

func ParseMetadataFromRaw(raw string) (entities.FileMetadata, error) {
	meta, err := helper.ParseMetadataJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata json: %w", err)
	}
	return meta, nil
}
