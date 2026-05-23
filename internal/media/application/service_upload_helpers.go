package application

// Upload helpers (stateless functions) extracted from service.go to stay within file-length limits.

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

func uploadToProvider(gw domain.MediaGateway, provider, objectKey, filename string, payload []byte, meta domain.RawMetadata) (domain.ProviderUploadResult, error) {
	return gw.UploadToProvider(context.Background(), provider, objectKey, filename, payload, meta)
}

func mergeProviderMetadataWithPrevious(gw domain.MediaGateway, upload domain.ProviderUploadResult, prev domain.RawMetadata) domain.RawMetadata {
	uploadedMeta := gw.NormalizeMetadata(upload.Metadata)
	merged := gw.NormalizeMetadata(uploadedMeta)
	for k, v := range prev {
		if _, ok := merged[k]; !ok {
			merged[k] = v
		}
	}
	return merged
}

type createUploadInputParams struct {
	gw                 domain.MediaGateway
	header             *multipart.FileHeader
	payload            []byte
	filename           string
	mime               string
	kind               string
	provider           string
	requestedObjectKey string
	uploaded           domain.ProviderUploadResult
	now                time.Time
}

func buildCreateEntityInput(p createUploadInputParams) domain.MediaUploadEntityInput {
	uploadedMeta := p.gw.NormalizeMetadata(p.uploaded.Metadata)
	merged := p.gw.NormalizeMetadata(uploadedMeta)
	isImage := p.gw.IsImageMIMEOrExt(p.mime, p.filename)
	return domain.MediaUploadEntityInput{
		Kind:          p.kind,
		Provider:      p.provider,
		Filename:      p.filename,
		ContentType:   p.mime,
		SizeBytes:     effectiveUploadSizeBytes(p.header.Size, p.payload, isImage),
		Payload:       p.payload,
		Uploaded:      p.uploaded,
		UploadedMeta:  merged,
		B2Bucket:      strings.TrimSpace(setting.MediaSetting.B2Bucket),
		CreatedAt:     p.now,
		UpdatedAt:     p.now,
		GenerateNewID: true,
	}
}

type updateUploadInputParams struct {
	prevFile  *domain.File
	kind      string
	provider  string
	filename  string
	mime      string
	sizeBytes int64
	payload   []byte
	uploaded  domain.ProviderUploadResult
	merged    domain.RawMetadata
}

func buildUpdateEntityInput(p updateUploadInputParams) domain.MediaUploadEntityInput {
	return domain.MediaUploadEntityInput{
		Kind:          p.kind,
		Provider:      p.provider,
		Filename:      p.filename,
		ContentType:   p.mime,
		SizeBytes:     p.sizeBytes,
		Payload:       p.payload,
		Uploaded:      p.uploaded,
		UploadedMeta:  p.merged,
		B2Bucket:      strings.TrimSpace(setting.MediaSetting.B2Bucket),
		CreatedAt:     p.prevFile.CreatedAt,
		UpdatedAt:     time.Now(),
		GenerateNewID: false,
		PreserveID:    p.prevFile.ID,
	}
}

func prepareCreateMultipartBody(gw domain.MediaGateway, req CreateFileInput, file multipart.File, fileHeader *multipart.FileHeader, remainingTotal *int64) (
	payload []byte, filename, mime string, kind string, provider string, objectKey string, err error,
) {
	payload, filename, mime, err = readMultipartPayloadLimited(file, fileHeader, remainingTotal)
	if err != nil {
		return
	}
	kind, kindInferred := gw.ResolveMediaKindFromServer(mime, filename)
	provider = gw.ResolveUploadProvider(kind, kindInferred)
	objectKey = gw.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)
	isImage := gw.IsImageMIMEOrExt(mime, filename)
	if err = rejectExecutableNonMedia(kind, isImage, filename, payload); err != nil {
		return
	}
	if isImage {
		var enc []byte
		var newMime, newName string
		enc, newMime, newName, err = encodeUploadToWebP(payload, filename)
		if err != nil {
			return
		}
		payload, mime, filename = enc, newMime, newName
		objectKey = gw.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)
	}
	return
}

func normalizeUpdateMultipartPayload(gw domain.MediaGateway, filename, mime string, payload []byte) (
	newPayload []byte, newFilename, newMime string, kind string, provider string, objectKey string, err error,
) {
	kind, kindInferred := gw.ResolveMediaKindFromServer(mime, filename)
	provider = gw.ResolveUploadProvider(kind, kindInferred)
	objectKey = gw.ResolveMediaUploadObjectKey("", filename, provider)
	isImage := gw.IsImageMIMEOrExt(mime, filename)
	if err = rejectExecutableNonMedia(kind, isImage, filename, payload); err != nil {
		return
	}
	if isImage {
		var enc []byte
		var encMime, encName string
		enc, encMime, encName, err = encodeUploadToWebP(payload, filename)
		if err != nil {
			return
		}
		newPayload, newMime, newFilename = enc, encMime, encName
		objectKey = gw.ResolveMediaUploadObjectKey("", newFilename, provider)
		return
	}
	newPayload, newMime, newFilename = payload, mime, filename
	return
}

func prepareCreatePartsSequential(gw domain.MediaGateway, req CreateFileInput, parts []domain.OpenedUploadPart, remaining *int64) ([]domain.PreparedCreatePart, error) {
	prepared := make([]domain.PreparedCreatePart, 0, len(parts))
	for _, part := range parts {
		payload, filename, mime, err := readMultipartPayloadLimited(part.File, part.Header, remaining)
		if err != nil {
			return nil, err
		}
		kind, kindInferred := gw.ResolveMediaKindFromServer(mime, filename)
		provider := gw.ResolveUploadProvider(kind, kindInferred)
		objectKey := gw.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)
		isImage := gw.IsImageMIMEOrExt(mime, filename)
		if err := rejectExecutableNonMedia(kind, isImage, filename, payload); err != nil {
			return nil, err
		}
		if isImage {
			enc, encMime, encName, encErr := encodeUploadToWebP(payload, filename)
			if encErr != nil {
				return nil, encErr
			}
			payload, mime, filename = enc, encMime, encName
			objectKey = gw.ResolveMediaUploadObjectKey(req.ObjectKey, filename, provider)
		}
		prepared = append(prepared, domain.PreparedCreatePart{
			Header: part.Header, Payload: payload, Filename: filename,
			Mime: mime, Kind: kind, Provider: provider, ObjectKey: objectKey,
		})
	}
	return prepared, nil
}

func prepareOptionalTailPrepared(gw domain.MediaGateway, createReq CreateFileInput, parts []domain.OpenedUploadPart, remaining *int64) ([]domain.PreparedCreatePart, error) {
	if len(parts) == 0 {
		return nil, nil
	}
	return prepareCreatePartsSequential(gw, createReq, parts, remaining)
}

func readMultipartPayloadLimited(file multipart.File, fileHeader *multipart.FileHeader, remainingTotal *int64) (payload []byte, filename, mime string, err error) {
	filename = strings.TrimSpace(fileHeader.Filename)
	mime = fileHeader.Header.Get("Content-Type")

	perPartCap := multipartPerPartCap(remainingTotal)
	if perPartCap <= 0 {
		return nil, filename, mime, apperrors.ErrMediaMultipartTotalTooLarge
	}
	if fileHeader.Size >= 0 && fileHeader.Size > constants.MaxMediaUploadFileBytes {
		return nil, filename, mime, apperrors.ErrFileExceedsMaxUploadSize
	}
	if fileHeader.Size >= 0 && fileHeader.Size > perPartCap {
		return nil, filename, mime, apperrors.ErrMediaMultipartTotalTooLarge
	}
	limited := limitReader(file, perPartCap+1)
	payload, err = readAll(limited)
	if err != nil {
		return nil, filename, mime, err
	}
	if int64(len(payload)) > constants.MaxMediaUploadFileBytes {
		return nil, filename, mime, apperrors.ErrFileExceedsMaxUploadSize
	}
	if int64(len(payload)) > perPartCap {
		return nil, filename, mime, apperrors.ErrMediaMultipartTotalTooLarge
	}
	if remainingTotal != nil {
		*remainingTotal -= int64(len(payload))
	}
	return payload, filename, mime, nil
}

func multipartPerPartCap(remainingTotal *int64) int64 {
	perPartCap := constants.MaxMediaUploadFileBytes
	if remainingTotal != nil && *remainingTotal < perPartCap {
		perPartCap = *remainingTotal
	}
	return perPartCap
}

func rejectExecutableNonMedia(kind string, isImage bool, filename string, payload []byte) error {
	if kind != constants.FileKindFile || isImage {
		return nil
	}
	head := payload
	if len(head) > 16 {
		head = head[:16]
	}
	if utils.IsExecutableUploadRejected(filename, head) {
		return apperrors.ErrExecutableUploadRejected
	}
	return nil
}

func effectiveUploadSizeBytes(headerSize int64, payload []byte, isImage bool) int64 {
	if isImage {
		return int64(len(payload))
	}
	if headerSize <= 0 {
		return int64(len(payload))
	}
	return headerSize
}

func encodeUploadToWebP(payload []byte, filename string) ([]byte, string, string, error) {
	utils.AcquireEncodeGate()
	encoded, newMime, encErr := utils.EncodeWebP(payload)
	utils.ReleaseEncodeGate()
	if encErr != nil {
		return nil, "", "", &domain.ProviderError{
			Code: apperrors.ImageEncodeBusy,
			Msg:  encErr.Error(),
			Err:  encErr,
		}
	}
	outName := filename
	if ext := filepath.Ext(filename); ext != "" {
		outName = strings.TrimSuffix(filename, ext) + ".webp"
	}
	return encoded, newMime, outName, nil
}

// ValidateBatchDeleteKeys performs the pure input validation for a batch-delete request
// (length limit, empty keys, duplicate keys) without touching the database or cloud clients.
func ValidateBatchDeleteKeys(keys []string) error {
	if len(keys) == 0 {
		return apperrors.ErrBatchDeleteEmptyKeys
	}
	if len(keys) > constants.MaxMediaBatchDelete {
		return apperrors.ErrMediaBatchDeleteTooManyIDs
	}
	_, err := dedupeBatchDeleteKeys(keys)
	return err
}

func dedupeBatchDeleteKeys(keys []string) ([]string, error) {
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			return nil, apperrors.ErrMediaObjectKeyRequired
		}
		if _, ok := seen[k]; ok {
			return nil, apperrors.ErrMediaDuplicateObjectKeysInBatchDelete
		}
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out, nil
}

func deleteUploadAttempt(gw domain.MediaGateway, provider, objectKey string, uploaded domain.ProviderUploadResult) {
	bunny := ""
	if uploaded.Metadata != nil {
		if v := uploaded.Metadata[domain.MediaMetaKeyBunnyVideoID]; v != nil {
			bunny = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if bunny == "" && uploaded.Metadata[domain.MediaMetaKeyVideoGUID] != nil {
			bunny = strings.TrimSpace(fmt.Sprintf("%v", uploaded.Metadata[domain.MediaMetaKeyVideoGUID]))
		}
	}
	_ = gw.DeleteStoredObject(context.Background(), objectKey, provider, bunny)
}

func scheduleParallelUpload(g *errgroup.Group, sem chan struct{}, fn func() error) {
	g.Go(func() error {
		sem <- struct{}{}
		defer func() { <-sem }()
		return fn()
	})
}

func runParallelProviderUpload(
	gw domain.MediaGateway,
	provider, objectKey, filename string,
	payload []byte,
) (domain.ProviderUploadResult, error) {
	if MediaUploadParallelStartProbe != nil {
		MediaUploadParallelStartProbe()
	}
	return uploadToProvider(gw, provider, objectKey, filename, payload, domain.RawMetadata{})
}

func recordParallelUploadIndex(mu *sync.Mutex, idx int, result domain.ProviderUploadResult, finished *[]int, results []domain.ProviderUploadResult) {
	mu.Lock()
	results[idx] = result
	*finished = append(*finished, idx)
	mu.Unlock()
}

func uploadPreparedCreatesParallel(gw domain.MediaGateway, prepared []domain.PreparedCreatePart) ([]domain.ProviderUploadResult, error) {
	if len(prepared) == 0 {
		return nil, nil
	}
	results := make([]domain.ProviderUploadResult, len(prepared))
	var mu sync.Mutex
	var finished []int

	g := new(errgroup.Group)
	sem := make(chan struct{}, constants.MaxConcurrentMediaUploadWorkers)
	for i := range prepared {
		i := i
		scheduleParallelUpload(g, sem, func() error {
			r, err := runParallelProviderUpload(gw, prepared[i].Provider, prepared[i].ObjectKey, prepared[i].Filename, prepared[i].Payload)
			if err != nil {
				return err
			}
			recordParallelUploadIndex(&mu, i, r, &finished, results)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		mu.Lock()
		idxs := append([]int(nil), finished...)
		mu.Unlock()
		for _, idx := range idxs {
			deleteUploadAttempt(gw, prepared[idx].Provider, prepared[idx].ObjectKey, results[idx])
		}
		return nil, err
	}
	return results, nil
}

func uploadBundleParallel(gw domain.MediaGateway, head *domain.PreparedUpdateHead, tail []domain.PreparedCreatePart) (domain.ProviderUploadResult, []domain.ProviderUploadResult, error) {
	var headResult domain.ProviderUploadResult
	tailResults := make([]domain.ProviderUploadResult, len(tail))

	var mu sync.Mutex
	var headFinished bool
	var tailFinished []int

	g := new(errgroup.Group)
	sem := make(chan struct{}, constants.MaxConcurrentMediaUploadWorkers)

	schedule := func(fn func() error) { scheduleParallelUpload(g, sem, fn) }

	if head != nil {
		schedule(func() error {
			r, err := runParallelProviderUpload(gw, head.Provider, head.ResolvedObjectKey, head.FilenameNorm, head.PayloadNorm)
			if err != nil {
				return err
			}
			mu.Lock()
			headResult = r
			headFinished = true
			mu.Unlock()
			return nil
		})
	}
	for i := range tail {
		i := i
		schedule(func() error {
			r, err := runParallelProviderUpload(gw, tail[i].Provider, tail[i].ObjectKey, tail[i].Filename, tail[i].Payload)
			if err != nil {
				return err
			}
			recordParallelUploadIndex(&mu, i, r, &tailFinished, tailResults)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if headFinished && head != nil {
			deleteUploadAttempt(gw, head.Provider, head.ResolvedObjectKey, headResult)
		}
		for _, idx := range tailFinished {
			deleteUploadAttempt(gw, tail[idx].Provider, tail[idx].ObjectKey, tailResults[idx])
		}
		return domain.ProviderUploadResult{}, nil, err
	}
	return headResult, tailResults, nil
}

var limitReader = func(r multipart.File, n int64) interface{ Read([]byte) (int, error) } {
	return &limitedReader{r: r, n: n}
}

type limitedReader struct {
	r interface{ Read([]byte) (int, error) }
	n int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, fmt.Errorf("read limit exceeded")
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	return n, err
}

var readAll = func(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf bytes.Buffer
	_, err := func() (int64, error) {
		p := make([]byte, 32*1024)
		var total int64
		for {
			n, err := r.Read(p)
			if n > 0 {
				buf.Write(p[:n])
				total += int64(n)
			}
			if err != nil {
				if err.Error() == "EOF" || strings.Contains(err.Error(), "EOF") {
					return total, nil
				}
				return total, err
			}
		}
	}()
	return buf.Bytes(), err
}

func isBunnyWebhookFinishStatus(status int) bool {
	return status == domain.BunnyFinished || status == domain.BunnyResolutionFinished
}

func isBunnyWebhookStatusSupported(status int) bool {
	switch status {
	case domain.BunnyQueued,
		domain.BunnyProcessing,
		domain.BunnyEncoding,
		domain.BunnyFinished,
		domain.BunnyResolutionFinished,
		domain.BunnyFailed,
		domain.BunnyPresignedUploadStarted,
		domain.BunnyPresignedUploadFinished,
		domain.BunnyPresignedUploadFailed,
		domain.BunnyCaptionsGenerated,
		domain.BunnyTitleOrDescriptionGenerated:
		return true
	default:
		return false
	}
}
