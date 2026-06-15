package application_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"

	mediaapp "mycourse-io-be/internal/media/application"
	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
)

type fakeFileRepo struct {
	rows          map[string]*domain.File
	getErr        error
	softDeleteErr error
	softDeleted   []string
}

func (f *fakeFileRepo) List(_ context.Context, _ domain.FileFilter) ([]domain.File, int64, error) {
	return nil, 0, nil
}

func (f *fakeFileRepo) GetByID(_ context.Context, _ string) (*domain.File, error) {
	return nil, apperrors.ErrNotFound
}

func (f *fakeFileRepo) GetByObjectKey(_ context.Context, objectKey string) (*domain.File, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if row, ok := f.rows[objectKey]; ok {
		return row, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeFileRepo) GetByBunnyVideoID(_ context.Context, _ string) (*domain.File, error) {
	return nil, apperrors.ErrNotFound
}

func (f *fakeFileRepo) ListBunnyVideoGUIDsWithMissingDuration(_ context.Context, _ int) ([]string, error) {
	return nil, nil
}

func (f *fakeFileRepo) UpsertByObjectKey(_ context.Context, _ *domain.File) error {
	return nil
}

func (f *fakeFileRepo) SaveWithRowVersionCheck(_ context.Context, _ *domain.File, _ int64) error {
	return nil
}

func (f *fakeFileRepo) SoftDeleteByObjectKey(_ context.Context, objectKey string) error {
	f.softDeleted = append(f.softDeleted, objectKey)
	return f.softDeleteErr
}

type deleteCall struct {
	objectKey string
	provider  string
	bunnyID   string
}

type fakeMediaGateway struct {
	defaultProviders map[string]string
	requireErr       error
	deleteErr        error
	deleteCalls      []deleteCall
}

func (f *fakeMediaGateway) RequireCloudReady() error { return f.requireErr }

func (f *fakeMediaGateway) DefaultMediaProvider(kind string) string {
	if p, ok := f.defaultProviders[kind]; ok {
		return p
	}
	return constants.FileProviderB2
}

func (f *fakeMediaGateway) BuildPublicURL(_, _ string) string { return "" }

func (f *fakeMediaGateway) NormalizeMetadata(in map[string]any) domain.RawMetadata {
	return domain.RawMetadata(in)
}

func (f *fakeMediaGateway) ParseMetadataFromRaw(_ string) (domain.RawMetadata, error) {
	return domain.RawMetadata{}, nil
}

func (f *fakeMediaGateway) DecodeLocalURLToken(_ string) (string, error) { return "", nil }

func (f *fakeMediaGateway) ResolveMediaKindFromServer(_, _ string) (string, bool) {
	return constants.FileKindFile, true
}

func (f *fakeMediaGateway) ResolveUploadProvider(_ string, _ bool) string {
	return constants.FileProviderB2
}

func (f *fakeMediaGateway) ResolveMediaUploadObjectKey(_, _, _ string) string { return "" }

func (f *fakeMediaGateway) IsImageMIMEOrExt(_, _ string) bool { return false }

func (f *fakeMediaGateway) UploadToProvider(
	_ context.Context,
	_,
	_,
	_ string,
	_ []byte,
	_ domain.RawMetadata,
) (domain.ProviderUploadResult, error) {
	return domain.ProviderUploadResult{}, nil
}

func (f *fakeMediaGateway) DeleteStoredObject(_ context.Context, objectKey, provider, bunnyVideoID string) error {
	f.deleteCalls = append(f.deleteCalls, deleteCall{objectKey: objectKey, provider: provider, bunnyID: bunnyVideoID})
	return f.deleteErr
}

func (f *fakeMediaGateway) BuildMediaFileEntityFromUpload(_ domain.MediaUploadEntityInput) *domain.File {
	return &domain.File{}
}

func (f *fakeMediaGateway) GetBunnyVideoByID(_ context.Context, _ string) (*domain.BunnyVideoDetail, error) {
	return nil, nil
}

func (f *fakeMediaGateway) BunnyStatusString(_ int) string { return "" }

func (f *fakeMediaGateway) ProfileImageFileAcceptable(_, _, _ string) bool { return true }

func (f *fakeMediaGateway) ShouldEnqueueSupersededCloudCleanup(_, _, _, _ string) bool { return false }

func (f *fakeMediaGateway) MergeMediaMetadataJSON(_ []byte, _ domain.RawMetadata) ([]byte, error) {
	return nil, nil
}

func (f *fakeMediaGateway) ApplyBunnyDetailToMetadata(_ domain.RawMetadata, _ *domain.BunnyVideoDetail, _, _ string) {
}

func (f *fakeMediaGateway) ApplyBunnyStreamFileColumns(_ *domain.File, _ *domain.BunnyVideoDetail, _, _ string) {
}

func (f *fakeMediaGateway) BuildTypedMetadata(
	_,
	_,
	_ string,
	_ int64,
	_ []byte,
	_ domain.RawMetadata,
) domain.UploadFileMetadata {
	return domain.UploadFileMetadata{}
}

func (f *fakeMediaGateway) ApplyTypedMetadataToRaw(_ domain.RawMetadata, _ domain.UploadFileMetadata) {
}

func (f *fakeMediaGateway) EffectiveBunnyThumbnailURL(_ *domain.BunnyVideoDetail) string { return "" }

func (f *fakeMediaGateway) BunnyWebhookSigningSecret() string { return "" }

func (f *fakeMediaGateway) IsBunnyWebhookSignatureValid(_ []byte, _, _, _, _ string) bool {
	return false
}

func (f *fakeMediaGateway) CollectMultipartFileHeaders(_ *multipart.Form) []*multipart.FileHeader {
	return nil
}

func (f *fakeMediaGateway) ValidateMultipartFileHeaders(_ []*multipart.FileHeader) error { return nil }

func (f *fakeMediaGateway) OpenUploadParts(_ []*multipart.FileHeader) ([]domain.OpenedUploadPart, error) {
	return nil, nil
}

func (f *fakeMediaGateway) CloseOpenedUploadParts(_ []domain.OpenedUploadPart) {}

func TestDeleteFile_UsesPersistedProviderForFileEvenWhenMetadataContainsBunnyID(t *testing.T) {
	repo := &fakeFileRepo{
		rows: map[string]*domain.File{
			"img.webp": {
				ObjectKey: "img.webp",
				Kind:      constants.FileKindFile,
				Provider:  constants.FileProviderB2,
			},
		},
	}
	gw := &fakeMediaGateway{
		defaultProviders: map[string]string{
			constants.FileKindFile:  constants.FileProviderBunny,
			constants.FileKindVideo: constants.FileProviderB2,
		},
	}
	svc := mediaapp.NewMediaService(repo, nil, nil, nil, gw)
	meta := domain.RawMetadata{domain.MediaMetaKeyBunnyVideoID: "query-guid"}

	if err := svc.DeleteFile(context.Background(), "img.webp", meta); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}
	if len(gw.deleteCalls) != 1 {
		t.Fatalf("DeleteStoredObject calls = %d, want 1", len(gw.deleteCalls))
	}
	if got := gw.deleteCalls[0].provider; got != constants.FileProviderB2 {
		t.Fatalf("provider = %q, want %q", got, constants.FileProviderB2)
	}
	if got := gw.deleteCalls[0].bunnyID; got != "query-guid" {
		t.Fatalf("bunnyID = %q, want %q", got, "query-guid")
	}
	if len(repo.softDeleted) != 1 || repo.softDeleted[0] != "img.webp" {
		t.Fatalf("softDeleted = %#v, want [img.webp]", repo.softDeleted)
	}
}

func TestDeleteFile_UsesPersistedVideoProviderAndBunnyIDWhenMetadataEmpty(t *testing.T) {
	repo := &fakeFileRepo{
		rows: map[string]*domain.File{
			"video-obj": {
				ObjectKey:    "video-obj",
				Kind:         constants.FileKindVideo,
				Provider:     constants.FileProviderBunny,
				BunnyVideoID: "row-guid",
			},
		},
	}
	gw := &fakeMediaGateway{
		defaultProviders: map[string]string{
			constants.FileKindFile:  constants.FileProviderB2,
			constants.FileKindVideo: constants.FileProviderB2,
		},
	}
	svc := mediaapp.NewMediaService(repo, nil, nil, nil, gw)

	if err := svc.DeleteFile(context.Background(), "video-obj", domain.RawMetadata{}); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}
	if len(gw.deleteCalls) != 1 {
		t.Fatalf("DeleteStoredObject calls = %d, want 1", len(gw.deleteCalls))
	}
	if got := gw.deleteCalls[0].provider; got != constants.FileProviderBunny {
		t.Fatalf("provider = %q, want %q", got, constants.FileProviderBunny)
	}
	if got := gw.deleteCalls[0].bunnyID; got != "row-guid" {
		t.Fatalf("bunnyID = %q, want %q", got, "row-guid")
	}
}

func TestDeleteFile_FallbackWhenRowMissing_PreservesLegacyInference(t *testing.T) {
	t.Run("without metadata guid -> file provider default", func(t *testing.T) {
		repo := &fakeFileRepo{getErr: apperrors.ErrNotFound}
		gw := &fakeMediaGateway{
			defaultProviders: map[string]string{
				constants.FileKindFile:  constants.FileProviderB2,
				constants.FileKindVideo: constants.FileProviderBunny,
			},
		}
		svc := mediaapp.NewMediaService(repo, nil, nil, nil, gw)

		if err := svc.DeleteFile(context.Background(), "missing-file", domain.RawMetadata{}); err != nil {
			t.Fatalf("DeleteFile returned error: %v", err)
		}
		if len(gw.deleteCalls) != 1 {
			t.Fatalf("DeleteStoredObject calls = %d, want 1", len(gw.deleteCalls))
		}
		if got := gw.deleteCalls[0].provider; got != constants.FileProviderB2 {
			t.Fatalf("provider = %q, want %q", got, constants.FileProviderB2)
		}
		if got := gw.deleteCalls[0].bunnyID; got != "" {
			t.Fatalf("bunnyID = %q, want empty", got)
		}
	})

	t.Run("with metadata guid -> video provider default", func(t *testing.T) {
		repo := &fakeFileRepo{getErr: apperrors.ErrNotFound}
		gw := &fakeMediaGateway{
			defaultProviders: map[string]string{
				constants.FileKindFile:  constants.FileProviderB2,
				constants.FileKindVideo: constants.FileProviderBunny,
			},
		}
		svc := mediaapp.NewMediaService(repo, nil, nil, nil, gw)
		meta := domain.RawMetadata{domain.MediaMetaKeyVideoGUID: "meta-guid"}

		if err := svc.DeleteFile(context.Background(), "missing-video", meta); err != nil {
			t.Fatalf("DeleteFile returned error: %v", err)
		}
		if len(gw.deleteCalls) != 1 {
			t.Fatalf("DeleteStoredObject calls = %d, want 1", len(gw.deleteCalls))
		}
		if got := gw.deleteCalls[0].provider; got != constants.FileProviderBunny {
			t.Fatalf("provider = %q, want %q", got, constants.FileProviderBunny)
		}
		if got := gw.deleteCalls[0].bunnyID; got != "meta-guid" {
			t.Fatalf("bunnyID = %q, want %q", got, "meta-guid")
		}
	})
}

func TestDeleteFile_ReturnsRepoErrorWhenLookupFailsUnexpectedly(t *testing.T) {
	wantErr := errors.New("db down")
	repo := &fakeFileRepo{getErr: wantErr}
	gw := &fakeMediaGateway{
		defaultProviders: map[string]string{
			constants.FileKindFile: constants.FileProviderB2,
		},
	}
	svc := mediaapp.NewMediaService(repo, nil, nil, nil, gw)

	err := svc.DeleteFile(context.Background(), "k", domain.RawMetadata{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(gw.deleteCalls) != 0 {
		t.Fatalf("DeleteStoredObject should not be called on repo lookup error, got %d call(s)", len(gw.deleteCalls))
	}
}
