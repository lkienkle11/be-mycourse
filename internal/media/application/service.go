// Package application contains the MEDIA bounded-context application/use-case layer.
// It orchestrates domain types and infra interfaces; it must not import delivery DTOs.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra" //nolint:depguard // application orchestrates infra utilities (BuildPublicURL, BuildTypedMetadata, cloud clients); TODO: inject as domain ports
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"

	"go.uber.org/zap"
)

// OrphanCleanupEnqueuer is implemented by the jobs layer (injected to avoid import cycle).
type OrphanCleanupEnqueuer interface {
	EnqueueSupersededPendingCleanup(objectKey, provider, bunnyVideoID string)
}

// CleanupCounters returns atomic cleanup-job counters.
type CleanupCounters interface {
	Deleted() uint64
	Failed() uint64
	Retried() uint64
}

// MediaService provides all media use-cases (file upload, update, delete, video webhook).
type MediaService struct {
	fileRepo    domain.FileRepository
	cleanupRepo domain.PendingCleanupRepository
	enqueuer    OrphanCleanupEnqueuer
	counters    CleanupCounters
}

// NewMediaService constructs a MediaService with required dependencies.
func NewMediaService(
	fileRepo domain.FileRepository,
	cleanupRepo domain.PendingCleanupRepository,
	enqueuer OrphanCleanupEnqueuer,
	counters CleanupCounters,
) *MediaService {
	return &MediaService{
		fileRepo:    fileRepo,
		cleanupRepo: cleanupRepo,
		enqueuer:    enqueuer,
		counters:    counters,
	}
}

// --- Probe (test hook) -------------------------------------------------------

// MediaUploadParallelStartProbe is optionally set by tests to observe concurrent upload worker starts.
var MediaUploadParallelStartProbe func()

// --- List / Get --------------------------------------------------------------

// ListFiles returns a paginated list of media files.
func (s *MediaService) ListFiles(ctx context.Context, filter domain.FileFilter) ([]domain.File, int64, error) {
	return s.fileRepo.List(ctx, filter)
}

// GetFile resolves a file by object key. If not in DB it synthesises a domain.File from infra defaults.
func (s *MediaService) GetFile(ctx context.Context, objectKey, kind string) (*domain.File, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, apperrors.ErrMediaObjectKeyRequired
	}
	f, err := s.fileRepo.GetByObjectKey(ctx, key)
	if err == nil {
		return f, nil
	}
	resolvedProvider := mediainfra.DefaultMediaProvider(kind)
	fileURL := mediainfra.BuildPublicURL(resolvedProvider, key)
	now := time.Now()
	return &domain.File{
		ID:        key,
		Kind:      kind,
		Provider:  resolvedProvider,
		Filename:  key,
		URL:       fileURL,
		OriginURL: fileURL,
		ObjectKey: key,
		Status:    constants.FileStatusReady,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// --- Create ------------------------------------------------------------------

// CreateFile uploads one multipart file and persists it.
func (s *MediaService) CreateFile(ctx context.Context, req CreateFileInput, file multipart.File, fileHeader *multipart.FileHeader) (*domain.File, error) {
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return nil, err
	}
	clients := mediainfra.Cloud
	payload, filename, mime, kind, provider, objectKey, err := prepareCreateMultipartBody(req, file, fileHeader, nil)
	if err != nil {
		return nil, err
	}
	uploaded, err := uploadToProvider(clients, provider, objectKey, filename, payload, domain.RawMetadata{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	input := buildCreateEntityInput(fileHeader, payload, filename, mime, kind, provider, req.ObjectKey, uploaded, now)
	return s.persistCreateRow(ctx, clients, input, payload)
}

// CreateFiles uploads multiple multipart files in parallel and persists them (all-or-nothing).
func (s *MediaService) CreateFiles(ctx context.Context, req CreateFileInput, parts []domain.OpenedUploadPart) ([]*domain.File, error) {
	if len(parts) == 0 {
		return nil, apperrors.ErrMediaFilesRequired
	}
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return nil, err
	}
	clients := mediainfra.Cloud
	remaining := constants.MaxMediaMultipartTotalBytes
	prepared, err := prepareCreatePartsSequential(req, parts, &remaining)
	if err != nil {
		return nil, err
	}
	uploaded, err := uploadPreparedCreatesParallel(clients, prepared)
	if err != nil {
		return nil, err
	}
	return s.persistPreparedCreates(ctx, clients, prepared, uploaded)
}

// --- Update ------------------------------------------------------------------

// UpdateFile replaces the media row at objectKey with a new upload.
func (s *MediaService) UpdateFile(ctx context.Context, objectKey string, req UpdateFileInput, file multipart.File, fileHeader *multipart.FileHeader) (*domain.File, error) {
	prevRow, err := s.loadUpdateTarget(ctx, objectKey, req)
	if err != nil {
		return nil, err
	}
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return nil, err
	}
	return s.runUpdateFileMultipartBody(ctx, mediainfra.Cloud, prevRow, req, file, fileHeader, nil)
}

// UpdateFileBundle updates the primary row and creates additional tail rows in one request.
func (s *MediaService) UpdateFileBundle(ctx context.Context, objectKey string, req UpdateFileInput, createReq CreateFileInput, parts []domain.OpenedUploadPart) ([]*domain.File, error) {
	prevRow, err := s.loadUpdateTarget(ctx, objectKey, req)
	if err != nil {
		return nil, err
	}
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return nil, err
	}
	clients := mediainfra.Cloud
	out := make([]*domain.File, len(parts))
	remaining := constants.MaxMediaMultipartTotalBytes

	skipHead, headPrep, err := s.prepareUpdateBundleHead(ctx, prevRow, req, parts[0], &remaining)
	if err != nil {
		return nil, err
	}
	tailPrepared, err := prepareOptionalTailPrepared(createReq, parts[1:], &remaining)
	if err != nil {
		return nil, err
	}
	if skipHead != nil {
		return s.finishUpdateBundleSkipHead(ctx, clients, out, tailPrepared, skipHead)
	}
	headUploaded, tailUploaded, err := uploadBundleParallel(clients, headPrep, tailPrepared)
	if err != nil {
		return nil, err
	}
	if err := s.persistBundleAfterUpload(ctx, clients, prevRow, headPrep, tailPrepared, headUploaded, tailUploaded, out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Delete ------------------------------------------------------------------

// DeleteFile soft-deletes the row and removes the cloud object.
func (s *MediaService) DeleteFile(ctx context.Context, objectKey string, metadata domain.RawMetadata) error {
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return err
	}
	clients := mediainfra.Cloud
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return apperrors.ErrMediaObjectKeyRequired
	}
	provider := mediainfra.DefaultMediaProvider(constants.FileKindFile)
	bunnyID := strings.TrimSpace(fmt.Sprintf("%v", metadata[domain.MediaMetaKeyVideoGUID]))
	if bunnyID == "" {
		bunnyID = strings.TrimSpace(fmt.Sprintf("%v", metadata[domain.MediaMetaKeyBunnyVideoID]))
	}
	if bunnyID != "" {
		provider = mediainfra.DefaultMediaProvider(constants.FileKindVideo)
	}
	if err := mediainfra.DeleteStoredObject(context.Background(), clients, key, provider, bunnyID); err != nil {
		return err
	}
	return s.fileRepo.SoftDeleteByObjectKey(ctx, key)
}

// DeleteFilesByObjectKeys batch-deletes up to MaxMediaBatchDelete rows (all-or-nothing).
func (s *MediaService) DeleteFilesByObjectKeys(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return apperrors.ErrBatchDeleteEmptyKeys
	}
	if len(keys) > constants.MaxMediaBatchDelete {
		return apperrors.ErrMediaBatchDeleteTooManyIDs
	}
	uniq, err := dedupeBatchDeleteKeys(keys)
	if err != nil {
		return err
	}
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return err
	}
	clients := mediainfra.Cloud
	rows := make([]*domain.File, 0, len(uniq))
	for _, key := range uniq {
		row, err := s.fileRepo.GetByObjectKey(ctx, key)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				return apperrors.ErrMediaFileNotFoundForObjectKey
			}
			return err
		}
		rows = append(rows, row)
	}
	for _, row := range rows {
		provider := row.Provider
		if provider == "" {
			provider = mediainfra.DefaultMediaProvider(row.Kind)
		}
		if err := mediainfra.DeleteStoredObject(context.Background(), clients, row.ObjectKey, provider, row.BunnyVideoID); err != nil {
			return err
		}
		if err := s.fileRepo.SoftDeleteByObjectKey(ctx, row.ObjectKey); err != nil {
			return err
		}
	}
	return nil
}

// --- Video / Webhook ---------------------------------------------------------

// GetVideoStatus queries the Bunny Stream API for a video's processing status.
func (s *MediaService) GetVideoStatus(ctx context.Context, videoGUID string) (*domain.VideoProviderStatus, error) {
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return nil, err
	}
	guid := strings.TrimSpace(videoGUID)
	if guid == "" {
		return nil, apperrors.ErrMediaVideoGUIDRequired
	}
	video, err := mediainfra.GetBunnyVideoByID(mediainfra.Cloud, ctx, guid)
	if err != nil {
		return nil, err
	}
	return &domain.VideoProviderStatus{
		Status: mediainfra.BunnyStatusString(video.Status),
	}, nil
}

// HandleBunnyVideoWebhook processes a Bunny Stream webhook event.
func (s *MediaService) HandleBunnyVideoWebhook(ctx context.Context, req BunnyWebhookInput) error {
	log := logger.FromContext(ctx).With(
		zap.String("component", "bunny_webhook"),
		zap.Int("status", req.Status),
		zap.String("video_guid", req.VideoGUID),
	)
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		log.Warn("bunny webhook: media cloud client not initialized", zap.Error(err))
		return err
	}
	if !isBunnyWebhookStatusSupported(req.Status) {
		log.Debug("bunny webhook: unsupported status, no-op")
		return nil
	}
	if !isBunnyWebhookFinishStatus(req.Status) {
		go func() { _ = s.markBunnyWebhookFailedStatus(ctx, req.VideoGUID, req.Status) }()
		return nil
	}
	return s.applyBunnyWebhookFinishedStatus(ctx, req.VideoGUID)
}

// --- Profile media -----------------------------------------------------------

// LoadValidatedProfileImageFile resolves fileID to a ready, raster-image media row.
func (s *MediaService) LoadValidatedProfileImageFile(ctx context.Context, fileID string) (*domain.File, error) {
	id := strings.TrimSpace(fileID)
	if id == "" {
		return nil, apperrors.ErrInvalidProfileMediaFile
	}
	row, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrInvalidProfileMediaFile
		}
		return nil, err
	}
	if row.Status != constants.FileStatusReady {
		return nil, apperrors.ErrInvalidProfileMediaFile
	}
	if !mediainfra.ProfileImageFileAcceptable(row.Kind, row.MimeType, row.Filename) {
		return nil, apperrors.ErrInvalidProfileMediaFile
	}
	return row, nil
}

// --- Cleanup metrics ---------------------------------------------------------

// PendingCloudCleanupCounters returns the current cleanup-job counter snapshot.
func (s *MediaService) PendingCloudCleanupCounters() (deleted, failed, retried uint64) {
	if s.counters == nil {
		return 0, 0, 0
	}
	return s.counters.Deleted(), s.counters.Failed(), s.counters.Retried()
}

// ============================================================================
// Internal helpers
// ============================================================================

func (s *MediaService) persistCreateRow(ctx context.Context, clients *mediainfra.CloudClients, input domain.MediaUploadEntityInput, payload []byte) (*domain.File, error) {
	entity := mediainfra.BuildMediaFileEntityFromUpload(input)
	entity.B2BucketName = strings.TrimSpace(setting.MediaSetting.B2Bucket)
	entity.ContentFingerprint = utils.ContentFingerprint(payload)

	if err := s.fileRepo.UpsertByObjectKey(ctx, entity); err != nil {
		_ = mediainfra.DeleteStoredObject(context.Background(), clients, entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}
	saved, err := s.fileRepo.GetByObjectKey(ctx, entity.ObjectKey)
	if err != nil {
		return entity, nil
	}
	return saved, nil
}

func (s *MediaService) persistUpdatedRow(ctx context.Context, clients *mediainfra.CloudClients, prevFile *domain.File, input domain.MediaUploadEntityInput, payload []byte, fp string) (*domain.File, error) {
	entity := mediainfra.BuildMediaFileEntityFromUpload(input)
	entity.B2BucketName = strings.TrimSpace(setting.MediaSetting.B2Bucket)
	entity.ContentFingerprint = fp

	if err := s.fileRepo.SaveWithRowVersionCheck(ctx, entity, prevFile.RowVersion); err != nil {
		_ = mediainfra.DeleteStoredObject(context.Background(), clients, entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}
	if mediainfra.ShouldEnqueueSupersededCloudCleanup(prevFile.ObjectKey, prevFile.BunnyVideoID, entity.ObjectKey, entity.BunnyVideoID) {
		if s.enqueuer != nil {
			s.enqueuer.EnqueueSupersededPendingCleanup(prevFile.ObjectKey, prevFile.Provider, prevFile.BunnyVideoID)
		}
	}
	saved, err := s.fileRepo.GetByID(ctx, prevFile.ID)
	if err != nil {
		return entity, nil
	}
	return saved, nil
}

func (s *MediaService) saveUnchangedFingerprintMetadata(ctx context.Context, prevFile *domain.File, filename string) (*domain.File, error) {
	merged := mediainfra.NormalizeMetadata(prevFile.RawMetadataMap())
	blob, err := mediainfra.MergeMediaMetadataJSON(prevFile.MetadataJSONBytes(), domain.RawMetadata{})
	if err != nil {
		return nil, err
	}
	updated := *prevFile
	updated.MetadataJSON = string(blob)
	updated.UpdatedAt = time.Now()
	if filename != "" {
		updated.Filename = filename
	}
	_ = merged
	if err := s.fileRepo.SaveWithRowVersionCheck(ctx, &updated, prevFile.RowVersion); err != nil {
		return nil, err
	}
	saved, err := s.fileRepo.GetByID(ctx, prevFile.ID)
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *MediaService) loadUpdateTarget(ctx context.Context, objectKey string, req UpdateFileInput) (*domain.File, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, apperrors.ErrMediaObjectKeyRequired
	}
	prevFile, err := s.fileRepo.GetByObjectKey(ctx, key)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrMediaFileNotFoundForObjectKey
		}
		return nil, err
	}
	if rid := strings.TrimSpace(req.ReuseMediaID); rid != "" && rid != prevFile.ID {
		return nil, apperrors.ErrMediaReuseMismatch
	}
	if req.ExpectedRowVersion != nil && *req.ExpectedRowVersion != prevFile.RowVersion {
		return nil, apperrors.ErrMediaOptimisticLock
	}
	return prevFile, nil
}

func (s *MediaService) runUpdateFileMultipartBody(ctx context.Context, clients *mediainfra.CloudClients, prevFile *domain.File, req UpdateFileInput, file multipart.File, fileHeader *multipart.FileHeader, remainingTotal *int64) (*domain.File, error) {
	prevRaw := domain.RawMetadata{}
	if raw := prevFile.RawMetadataMap(); raw != nil {
		prevRaw = raw
	}

	payload, filename, mime, err := readMultipartPayloadLimited(file, fileHeader, remainingTotal)
	if err != nil {
		return nil, err
	}
	fp := utils.ContentFingerprint(payload)
	if req.SkipUploadIfUnchanged && prevFile.ContentFingerprint != "" && fp == prevFile.ContentFingerprint {
		return s.saveUnchangedFingerprintMetadata(ctx, prevFile, filename)
	}

	payload, filename, mime, kind, provider, resolvedObjectKey, err := normalizeUpdateMultipartPayload(filename, mime, payload)
	if err != nil {
		return nil, err
	}
	isImage := mediainfra.IsImageMIMEOrExt(mime, filename)

	uploaded, err := uploadToProvider(clients, provider, resolvedObjectKey, filename, payload, domain.RawMetadata{})
	if err != nil {
		return nil, err
	}

	merged := mergeProviderMetadataWithPrevious(uploaded, prevRaw)
	sizeBytes := effectiveUploadSizeBytes(fileHeader.Size, payload, isImage)
	input := buildUpdateEntityInput(prevFile, kind, provider, filename, mime, sizeBytes, payload, uploaded, merged)
	return s.persistUpdatedRow(ctx, clients, prevFile, input, payload, fp)
}

func (s *MediaService) prepareUpdateBundleHead(ctx context.Context, prevFile *domain.File, req UpdateFileInput, part domain.OpenedUploadPart, remaining *int64) (*domain.File, *domain.PreparedUpdateHead, error) {
	prevRaw := domain.RawMetadata{}
	if raw := prevFile.RawMetadataMap(); raw != nil {
		prevRaw = raw
	}
	payload, filename, mime, err := readMultipartPayloadLimited(part.File, part.Header, remaining)
	if err != nil {
		return nil, nil, err
	}
	fp := utils.ContentFingerprint(payload)
	if req.SkipUploadIfUnchanged && prevFile.ContentFingerprint != "" && fp == prevFile.ContentFingerprint {
		skipped, err := s.saveUnchangedFingerprintMetadata(ctx, prevFile, filename)
		if err != nil {
			return nil, nil, err
		}
		return skipped, nil, nil
	}
	payloadNorm, filenameNorm, mimeNorm, kind, provider, resolvedObjectKey, err := normalizeUpdateMultipartPayload(filename, mime, payload)
	if err != nil {
		return nil, nil, err
	}
	_ = prevRaw
	return nil, &domain.PreparedUpdateHead{
		Header: part.Header, Payload: payload, Filename: filename, Mime: mime,
		Fingerprint: fp, PayloadNorm: payloadNorm, FilenameNorm: filenameNorm,
		MimeNorm: mimeNorm, Kind: kind, Provider: provider, ResolvedObjectKey: resolvedObjectKey,
	}, nil
}

func (s *MediaService) persistPreparedCreates(ctx context.Context, clients *mediainfra.CloudClients, prepared []domain.PreparedCreatePart, uploaded []domain.ProviderUploadResult) ([]*domain.File, error) {
	if len(prepared) == 0 {
		return nil, nil
	}
	out := make([]*domain.File, len(prepared))
	now := time.Now()
	for i := range prepared {
		input := buildCreateEntityInput(prepared[i].Header, prepared[i].Payload, prepared[i].Filename, prepared[i].Mime, prepared[i].Kind, prepared[i].Provider, prepared[i].ObjectKey, uploaded[i], now)
		ent, err := s.persistCreateRow(ctx, clients, input, prepared[i].Payload)
		if err != nil {
			s.rollbackCreatedRows(ctx, clients, out[:i])
			return nil, err
		}
		out[i] = ent
	}
	return out, nil
}

func (s *MediaService) rollbackCreatedRows(ctx context.Context, clients *mediainfra.CloudClients, rows []*domain.File) {
	for _, ent := range rows {
		if ent == nil {
			continue
		}
		_ = mediainfra.DeleteStoredObject(context.Background(), clients, ent.ObjectKey, ent.Provider, ent.BunnyVideoID)
		_ = s.fileRepo.SoftDeleteByObjectKey(ctx, ent.ObjectKey)
	}
}

func (s *MediaService) finishUpdateBundleSkipHead(ctx context.Context, clients *mediainfra.CloudClients, out []*domain.File, tailPrepared []domain.PreparedCreatePart, skipHead *domain.File) ([]*domain.File, error) {
	out[0] = skipHead
	if len(tailPrepared) == 0 {
		return out[:1], nil
	}
	uploadedTail, err := uploadPreparedCreatesParallel(clients, tailPrepared)
	if err != nil {
		return nil, err
	}
	tailEntities, err := s.persistPreparedCreates(ctx, clients, tailPrepared, uploadedTail)
	if err != nil {
		return nil, err
	}
	for i := range tailEntities {
		out[1+i] = tailEntities[i]
	}
	return out, nil
}

func (s *MediaService) persistBundleAfterUpload(ctx context.Context, clients *mediainfra.CloudClients, prevFile *domain.File, headPrep *domain.PreparedUpdateHead, tailPrepared []domain.PreparedCreatePart, headUploaded domain.ProviderUploadResult, tailUploaded []domain.ProviderUploadResult, out []*domain.File) error {
	tailEntities, err := s.persistPreparedCreates(ctx, clients, tailPrepared, tailUploaded)
	if err != nil {
		deleteUploadAttempt(clients, headPrep.Provider, headPrep.ResolvedObjectKey, headUploaded)
		return err
	}
	prevRaw := domain.RawMetadata{}
	if raw := prevFile.RawMetadataMap(); raw != nil {
		prevRaw = raw
	}
	merged := mergeProviderMetadataWithPrevious(headUploaded, prevRaw)
	isImage := mediainfra.IsImageMIMEOrExt(headPrep.MimeNorm, headPrep.FilenameNorm)
	sizeBytes := effectiveUploadSizeBytes(headPrep.Header.Size, headPrep.PayloadNorm, isImage)
	input := buildUpdateEntityInput(prevFile, headPrep.Kind, headPrep.Provider, headPrep.FilenameNorm, headPrep.MimeNorm, sizeBytes, headPrep.PayloadNorm, headUploaded, merged)
	headEntity, err := s.persistUpdatedRow(ctx, clients, prevFile, input, headPrep.PayloadNorm, headPrep.Fingerprint)
	if err != nil {
		s.rollbackCreatedRows(ctx, clients, tailEntities)
		deleteUploadAttempt(clients, headPrep.Provider, headPrep.ResolvedObjectKey, headUploaded)
		return err
	}
	out[0] = headEntity
	for i := range tailEntities {
		out[1+i] = tailEntities[i]
	}
	return nil
}

// --- Video webhook helpers ---------------------------------------------------

func (s *MediaService) applyBunnyWebhookFinishedStatus(ctx context.Context, videoGUID string) error {
	log := logger.FromContext(ctx).With(
		zap.String("component", "bunny_webhook"),
		zap.String("video_guid", videoGUID),
	)
	trimmedGUID := strings.TrimSpace(videoGUID)
	video, err := mediainfra.GetBunnyVideoByID(mediainfra.Cloud, ctx, trimmedGUID)
	if err != nil {
		log.Warn("bunny webhook: GetBunnyVideoByID failed", zap.Error(err))
		return err
	}
	row, err := s.fileRepo.GetByBunnyVideoID(ctx, trimmedGUID)
	if err != nil {
		log.Debug("bunny webhook: no local media row, skipping", zap.Error(err))
		return nil
	}

	// Merge Bunny telemetry into the existing metadata map. We do NOT use
	// raw[k] = video.Field directly because Bunny's response often omits
	// fields (e.g. bitrate, audioCodec) and writing zero values would
	// destroy data that earlier webhook calls or the upload path already
	// populated. ApplyBunnyDetailToMetadata only writes non-zero fields.
	raw := row.RawMetadataMap()
	if raw == nil {
		raw = domain.RawMetadata{}
	}
	streamBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
	libID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	mediainfra.ApplyBunnyDetailToMetadata(raw, video, libID, streamBase)
	typed := mediainfra.BuildTypedMetadata(row.Kind, row.MimeType, row.Filename, row.SizeBytes, nil, raw)
	mediainfra.ApplyTypedMetadataToRaw(raw, typed)

	blob, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	row.MetadataJSON = string(blob)
	row.Status = constants.FileStatusReady
	if thumb := mediainfra.EffectiveBunnyThumbnailURL(video); thumb != "" {
		row.ThumbnailURL = thumb
	}
	if dur := int64(typed.DurationSeconds); dur > 0 {
		row.Duration = dur
	}

	if err := s.fileRepo.UpsertByObjectKey(ctx, row); err != nil {
		log.Warn("bunny webhook: upsert failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *MediaService) markBunnyWebhookFailedStatus(ctx context.Context, videoGUID string, status int) error {
	if status != domain.BunnyFailed && status != domain.BunnyPresignedUploadFailed {
		return nil
	}
	row, err := s.fileRepo.GetByBunnyVideoID(ctx, strings.TrimSpace(videoGUID))
	if err != nil {
		return err
	}
	row.Status = constants.FileStatusFailed
	return s.fileRepo.UpsertByObjectKey(ctx, row)
}

// Upload helper functions are in service_upload_helpers.go (same package).
