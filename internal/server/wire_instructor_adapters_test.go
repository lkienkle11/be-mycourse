package server

import (
	"context"
	"errors"
	"testing"

	"mycourse-io-be/internal/instructor/domain"
	mediadomain "mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
)

type stubMediaFileRepo struct {
	byID map[string]*mediadomain.File
}

func (s *stubMediaFileRepo) List(context.Context, mediadomain.FileFilter) ([]mediadomain.File, int64, error) {
	return nil, 0, nil
}

func (s *stubMediaFileRepo) GetByID(_ context.Context, id string) (*mediadomain.File, error) {
	f, ok := s.byID[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return f, nil
}

func (s *stubMediaFileRepo) GetByObjectKey(context.Context, string) (*mediadomain.File, error) {
	return nil, apperrors.ErrNotFound
}

func (s *stubMediaFileRepo) GetByBunnyVideoID(context.Context, string) (*mediadomain.File, error) {
	return nil, apperrors.ErrNotFound
}

func (s *stubMediaFileRepo) ListBunnyVideoGUIDsWithMissingDuration(context.Context, int) ([]string, error) {
	return nil, nil
}

func (s *stubMediaFileRepo) UpsertByObjectKey(context.Context, *mediadomain.File) error {
	return nil
}

func (s *stubMediaFileRepo) SaveWithRowVersionCheck(context.Context, *mediadomain.File, int64) error {
	return nil
}

func (s *stubMediaFileRepo) SoftDeleteByObjectKey(context.Context, string) error {
	return nil
}

func assertValidatePDF(t *testing.T, fileID string, file *mediadomain.File, missing bool, wantErr error) {
	t.Helper()
	byID := map[string]*mediadomain.File{}
	if !missing {
		byID[fileID] = file
	}
	v := &instructorProfileMediaValidator{files: &stubMediaFileRepo{byID: byID}}
	err := v.validatePDF(context.Background(), fileID)
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Fatalf("validatePDF() err = %v, want %v", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("validatePDF() unexpected err = %v", err)
	}
}

func TestInstructorProfileMediaValidator_validatePDF(t *testing.T) {
	t.Parallel()

	const fileID = "00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name    string
		file    *mediadomain.File
		missing bool
		wantErr error
	}{
		{
			name: "accepts application/pdf",
			file: &mediadomain.File{
				ID: fileID, Status: constants.FileStatusReady, MimeType: "application/pdf",
			},
		},
		{
			name: "rejects word document mime",
			file: &mediadomain.File{
				ID: fileID, Status: constants.FileStatusReady,
				MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				Filename: "cv.docx",
			},
			wantErr: apperrors.ErrInvalidProfileMediaFile,
		},
		{
			name: "rejects pdf extension without pdf mime",
			file: &mediadomain.File{
				ID: fileID, Status: constants.FileStatusReady,
				MimeType: "application/octet-stream", Filename: "cv.pdf",
			},
			wantErr: apperrors.ErrInvalidProfileMediaFile,
		},
		{name: "rejects missing file", missing: true, wantErr: apperrors.ErrInvalidProfileMediaFile},
		{
			name: "rejects non-ready status",
			file: &mediadomain.File{
				ID: fileID, Status: constants.FileStatusFailed, MimeType: "application/pdf",
			},
			wantErr: apperrors.ErrInvalidProfileMediaFile,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertValidatePDF(t, fileID, tc.file, tc.missing, tc.wantErr)
		})
	}
}

func TestInstructorProfileMediaValidator_ValidateProfilePayload_CVRequiredMime(t *testing.T) {
	t.Parallel()

	const fileID = "00000000-0000-0000-0000-000000000002"
	v := &instructorProfileMediaValidator{files: &stubMediaFileRepo{byID: map[string]*mediadomain.File{
		fileID: {ID: fileID, Status: constants.FileStatusReady, MimeType: "application/pdf"},
	}}}

	if err := v.ValidateProfilePayload(context.Background(), domain.ProfilePayload{CVFileID: fileID}); err != nil {
		t.Fatalf("ValidateProfilePayload() err = %v", err)
	}
}
