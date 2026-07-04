// Package application contains the MEDIA bounded-context application/use-case layer.
// It orchestrates domain types and infra interfaces; it must not import delivery DTOs.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/timex"
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
	gw          domain.MediaGateway
}

// NewMediaService constructs a MediaService with required dependencies.
func NewMediaService(
	fileRepo domain.FileRepository,
	cleanupRepo domain.PendingCleanupRepository,
	enqueuer OrphanCleanupEnqueuer,
	counters CleanupCounters,
	gw domain.MediaGateway,
) *MediaService {
	return &MediaService{
		fileRepo:    fileRepo,
		cleanupRepo: cleanupRepo,
		enqueuer:    enqueuer,
		counters:    counters,
		gw:          gw,
	}
}

// --- Probe (test hook) -------------------------------------------------------

// MediaUploadParallelStartProbe is optionally set by tests to observe concurrent upload worker starts.
var MediaUploadParallelStartProbe func()

// --- List / Get --------------------------------------------------------------

// ListFiles returns a paginated list of media files scoped to the viewer.
func (s *MediaService) ListFiles(ctx context.Context, filter domain.FileFilter) ([]domain.File, int64, error) {
	if strings.TrimSpace(filter.ViewerUserID) == "" {
		return nil, 0, apperrors.ErrMediaAccessDenied
	}
	files, total, err := s.fileRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	s.scheduleVideoDurationRefresh(context.WithoutCancel(ctx), files)
	return files, total, nil
}

// GetFile resolves a file by object key. Requires an active DB row; no storage URL fallback.
func (s *MediaService) GetFile(ctx context.Context, objectKey, kind, viewerUserID string) (*domain.File, error) {
	_ = kind
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil, apperrors.ErrMediaObjectKeyRequired
	}
	f, err := s.fileRepo.GetByObjectKey(ctx, key)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrMediaFileNotFoundForObjectKey
		}
		return nil, err
	}
	if !canViewMediaFile(f, viewerUserID) {
		return nil, apperrors.ErrMediaAccessDenied
	}
	s.scheduleVideoDurationRefresh(context.WithoutCancel(ctx), []domain.File{*f})
	return f, nil
}

// --- Create ------------------------------------------------------------------

// CreateFiles uploads multiple multipart files in parallel and persists them (all-or-nothing).
func (s *MediaService) CreateFiles(ctx context.Context, req CreateFileInput, parts []domain.OpenedUploadPart) ([]*domain.File, error) {
	if len(parts) == 0 {
		return nil, apperrors.ErrMediaFilesRequired
	}
	if err := s.gw.RequireCloudReady(); err != nil {
		return nil, err
	}
	req.Visibility = normalizeMediaVisibility(req.Visibility)
	remaining := constants.MaxMediaMultipartTotalBytes
	prepared, err := prepareCreatePartsSequential(s.gw, req, parts, &remaining)
	if err != nil {
		return nil, err
	}
	uploaded, err := uploadPreparedCreatesParallel(s.gw, prepared)
	if err != nil {
		return nil, err
	}
	return s.persistPreparedCreates(ctx, req, prepared, uploaded)
}

// --- Update ------------------------------------------------------------------

// UpdateFileBundle updates the primary row and creates additional tail rows in one request.
func (s *MediaService) UpdateFileBundle(ctx context.Context, objectKey string, req UpdateFileInput, createReq CreateFileInput, parts []domain.OpenedUploadPart, viewerUserID string) ([]*domain.File, error) {
	prevRow, err := s.loadUpdateTarget(ctx, objectKey, req, viewerUserID)
	if err != nil {
		return nil, err
	}
	if err := s.gw.RequireCloudReady(); err != nil {
		return nil, err
	}
	out := make([]*domain.File, len(parts))
	remaining := constants.MaxMediaMultipartTotalBytes

	skipHead, headPrep, err := s.prepareUpdateBundleHead(ctx, prevRow, req, parts[0], &remaining)
	if err != nil {
		return nil, err
	}
	tailPrepared, err := prepareOptionalTailPrepared(s.gw, createReq, parts[1:], &remaining)
	if err != nil {
		return nil, err
	}
	if skipHead != nil {
		return s.finishUpdateBundleSkipHead(ctx, createReq, out, tailPrepared, skipHead)
	}
	headUploaded, tailUploaded, err := uploadBundleParallel(s.gw, headPrep, tailPrepared)
	if err != nil {
		return nil, err
	}
	if err := s.persistBundleAfterUpload(ctx, createReq, prevRow, headPrep, tailPrepared, headUploaded, tailUploaded, out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Delete ------------------------------------------------------------------

// DeleteFile soft-deletes the row and removes the cloud object.
func (s *MediaService) DeleteFile(ctx context.Context, objectKey string, metadata domain.RawMetadata, viewerUserID string) error {
	if err := s.gw.RequireCloudReady(); err != nil {
		return err
	}
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return apperrors.ErrMediaObjectKeyRequired
	}
	bunnyID := utils.StringFromRaw(metadata, domain.MediaMetaKeyVideoGUID)
	if bunnyID == "" {
		bunnyID = utils.StringFromRaw(metadata, domain.MediaMetaKeyBunnyVideoID)
	}
	provider := s.gw.DefaultMediaProvider(constants.FileKindFile)

	row, err := s.fileRepo.GetByObjectKey(ctx, key)
	switch {
	case err == nil:
		if !canMutateMediaFile(row, viewerUserID) {
			return apperrors.ErrMediaAccessDenied
		}
		if rowProvider := strings.TrimSpace(row.Provider); rowProvider != "" {
			provider = rowProvider
		} else {
			rowKind := strings.TrimSpace(row.Kind)
			if rowKind == "" {
				rowKind = constants.FileKindFile
			}
			provider = s.gw.DefaultMediaProvider(rowKind)
		}
		if rowBunnyID := strings.TrimSpace(row.BunnyVideoID); rowBunnyID != "" {
			bunnyID = rowBunnyID
		}
	case errors.Is(err, apperrors.ErrNotFound):
		// Keep legacy behavior when DB row is absent: infer video deletion only from metadata.
		if bunnyID != "" {
			provider = s.gw.DefaultMediaProvider(constants.FileKindVideo)
		}
	default:
		return err
	}

	if err := s.gw.DeleteStoredObject(context.Background(), key, provider, bunnyID); err != nil {
		return err
	}
	return s.fileRepo.SoftDeleteByObjectKey(ctx, key)
}

// DeleteFilesByObjectKeys batch-deletes up to MaxMediaBatchDelete rows (all-or-nothing).
func (s *MediaService) DeleteFilesByObjectKeys(ctx context.Context, keys []string, viewerUserID string) error {
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
	if err := s.gw.RequireCloudReady(); err != nil {
		return err
	}
	rows := make([]*domain.File, 0, len(uniq))
	for _, key := range uniq {
		row, err := s.fileRepo.GetByObjectKey(ctx, key)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				return apperrors.ErrMediaFileNotFoundForObjectKey
			}
			return err
		}
		if !canMutateMediaFile(row, viewerUserID) {
			return apperrors.ErrMediaAccessDenied
		}
		rows = append(rows, row)
	}
	for _, row := range rows {
		provider := row.Provider
		if provider == "" {
			provider = s.gw.DefaultMediaProvider(row.Kind)
		}
		if err := s.gw.DeleteStoredObject(context.Background(), row.ObjectKey, provider, row.BunnyVideoID); err != nil {
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
	if err := s.gw.RequireCloudReady(); err != nil {
		return nil, err
	}
	guid := strings.TrimSpace(videoGUID)
	if guid == "" {
		return nil, apperrors.ErrMediaVideoGUIDRequired
	}
	video, err := s.gw.GetBunnyVideoByID(ctx, guid)
	if err != nil {
		return nil, err
	}
	return &domain.VideoProviderStatus{
		Status: s.gw.BunnyStatusString(video.Status),
	}, nil
}

// HandleBunnyVideoWebhook processes a Bunny Stream webhook event.
func (s *MediaService) HandleBunnyVideoWebhook(ctx context.Context, req BunnyWebhookInput) error {
	log := logger.FromContext(ctx).With(
		zap.String("component", "bunny_webhook"),
		zap.Int("status", req.Status),
		zap.String("video_guid", req.VideoGUID),
	)
	if err := s.gw.RequireCloudReady(); err != nil {
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
	if !s.gw.ProfileImageFileAcceptable(row.Kind, row.MimeType, row.Filename) {
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

func (s *MediaService) persistCreateRow(ctx context.Context, input domain.MediaUploadEntityInput, payload []byte) (*domain.File, error) {
	entity := s.gw.BuildMediaFileEntityFromUpload(input)
	entity.R2BucketName = strings.TrimSpace(setting.MediaSetting.R2.Bucket)
	entity.ContentFingerprint = utils.ContentFingerprint(payload)

	if err := s.fileRepo.UpsertByObjectKey(ctx, entity); err != nil {
		_ = s.gw.DeleteStoredObject(context.Background(), entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}
	saved, err := s.fileRepo.GetByObjectKey(ctx, entity.ObjectKey)
	if err != nil {
		return entity, nil
	}
	return saved, nil
}

func (s *MediaService) persistUpdatedRow(ctx context.Context, prevFile *domain.File, input domain.MediaUploadEntityInput, payload []byte, fp string) (*domain.File, error) {
	entity := s.gw.BuildMediaFileEntityFromUpload(input)
	entity.R2BucketName = strings.TrimSpace(setting.MediaSetting.R2.Bucket)
	entity.ContentFingerprint = fp

	if err := s.fileRepo.SaveWithRowVersionCheck(ctx, entity, prevFile.RowVersion); err != nil {
		_ = s.gw.DeleteStoredObject(context.Background(), entity.ObjectKey, entity.Provider, entity.BunnyVideoID)
		return nil, err
	}
	if s.gw.ShouldEnqueueSupersededCloudCleanup(prevFile.ObjectKey, prevFile.BunnyVideoID, entity.ObjectKey, entity.BunnyVideoID) {
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
	merged := s.gw.NormalizeMetadata(prevFile.RawMetadataMap())
	blob, err := s.gw.MergeMediaMetadataJSON(prevFile.MetadataJSONBytes(), domain.RawMetadata{})
	if err != nil {
		return nil, err
	}
	updated := *prevFile
	updated.MetadataJSON = string(blob)
	updated.UpdatedAt = timex.NowUnix()
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

func (s *MediaService) loadUpdateTarget(ctx context.Context, objectKey string, req UpdateFileInput, viewerUserID string) (*domain.File, error) {
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
	if !canMutateMediaFile(prevFile, viewerUserID) {
		return nil, apperrors.ErrMediaAccessDenied
	}
	if rid := strings.TrimSpace(req.ReuseMediaID); rid != "" && rid != prevFile.ID {
		return nil, apperrors.ErrMediaReuseMismatch
	}
	if req.ExpectedRowVersion != nil && *req.ExpectedRowVersion != prevFile.RowVersion {
		return nil, apperrors.ErrMediaOptimisticLock
	}
	return prevFile, nil
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
	payloadNorm, filenameNorm, mimeNorm, kind, provider, resolvedObjectKey, err := normalizeUpdateMultipartPayload(s.gw, filename, mime, payload)
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

func (s *MediaService) persistPreparedCreates(ctx context.Context, req CreateFileInput, prepared []domain.PreparedCreatePart, uploaded []domain.ProviderUploadResult) ([]*domain.File, error) {
	if len(prepared) == 0 {
		return nil, nil
	}
	out := make([]*domain.File, len(prepared))
	now := timex.NowUnix()
	for i := range prepared {
		input := buildCreateEntityInput(createUploadInputParams{
			gw: s.gw, header: prepared[i].Header, payload: prepared[i].Payload,
			filename: prepared[i].Filename, mime: prepared[i].Mime, kind: prepared[i].Kind,
			provider: prepared[i].Provider, requestedObjectKey: prepared[i].ObjectKey, uploaded: uploaded[i], now: now,
			userCode: req.UserCode, userID: req.UserID, visibility: req.Visibility,
		})
		ent, err := s.persistCreateRow(ctx, input, prepared[i].Payload)
		if err != nil {
			s.rollbackCreatedRows(ctx, out[:i])
			return nil, err
		}
		out[i] = ent
	}
	return out, nil
}

func (s *MediaService) rollbackCreatedRows(ctx context.Context, rows []*domain.File) {
	for _, ent := range rows {
		if ent == nil {
			continue
		}
		_ = s.gw.DeleteStoredObject(context.Background(), ent.ObjectKey, ent.Provider, ent.BunnyVideoID)
		_ = s.fileRepo.SoftDeleteByObjectKey(ctx, ent.ObjectKey)
	}
}

func (s *MediaService) finishUpdateBundleSkipHead(ctx context.Context, createReq CreateFileInput, out []*domain.File, tailPrepared []domain.PreparedCreatePart, skipHead *domain.File) ([]*domain.File, error) {
	out[0] = skipHead
	if len(tailPrepared) == 0 {
		return out[:1], nil
	}
	uploadedTail, err := uploadPreparedCreatesParallel(s.gw, tailPrepared)
	if err != nil {
		return nil, err
	}
	tailEntities, err := s.persistPreparedCreates(ctx, createReq, tailPrepared, uploadedTail)
	if err != nil {
		return nil, err
	}
	for i := range tailEntities {
		out[1+i] = tailEntities[i]
	}
	return out, nil
}

func (s *MediaService) persistBundleAfterUpload(ctx context.Context, createReq CreateFileInput, prevFile *domain.File, headPrep *domain.PreparedUpdateHead, tailPrepared []domain.PreparedCreatePart, headUploaded domain.ProviderUploadResult, tailUploaded []domain.ProviderUploadResult, out []*domain.File) error {
	tailEntities, err := s.persistPreparedCreates(ctx, createReq, tailPrepared, tailUploaded)
	if err != nil {
		deleteUploadAttempt(s.gw, headPrep.Provider, headPrep.ResolvedObjectKey, headUploaded)
		return err
	}
	prevRaw := domain.RawMetadata{}
	if raw := prevFile.RawMetadataMap(); raw != nil {
		prevRaw = raw
	}
	merged := mergeProviderMetadataWithPrevious(s.gw, headUploaded, prevRaw)
	isImage := s.gw.IsImageMIMEOrExt(headPrep.MimeNorm, headPrep.FilenameNorm)
	sizeBytes := effectiveUploadSizeBytes(headPrep.Header.Size, headPrep.PayloadNorm, isImage)
	input := buildUpdateEntityInput(updateUploadInputParams{
		prevFile: prevFile, kind: headPrep.Kind, provider: headPrep.Provider,
		filename: headPrep.FilenameNorm, mime: headPrep.MimeNorm, sizeBytes: sizeBytes,
		payload: headPrep.PayloadNorm, uploaded: headUploaded, merged: merged,
	})
	headEntity, err := s.persistUpdatedRow(ctx, prevFile, input, headPrep.PayloadNorm, headPrep.Fingerprint)
	if err != nil {
		s.rollbackCreatedRows(ctx, tailEntities)
		deleteUploadAttempt(s.gw, headPrep.Provider, headPrep.ResolvedObjectKey, headUploaded)
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
	video, err := s.gw.GetBunnyVideoByID(ctx, trimmedGUID)
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
	s.gw.ApplyBunnyDetailToMetadata(raw, video, libID, streamBase)
	s.gw.ApplyBunnyStreamFileColumns(row, video, libID, streamBase)
	typed := s.gw.BuildTypedMetadata(row.Kind, row.MimeType, row.Filename, row.SizeBytes, nil, raw)
	s.gw.ApplyTypedMetadataToRaw(raw, typed)

	blob, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	row.MetadataJSON = string(blob)
	row.Status = constants.FileStatusReady
	if dur := videoDurationSecondsFromTelemetry(typed.DurationSeconds, video.Length); dur > 0 {
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
