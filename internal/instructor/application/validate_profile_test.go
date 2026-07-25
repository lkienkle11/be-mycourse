package application

import (
	"context"
	stderrors "errors"
	"strconv"
	"strings"
	"testing"

	"mycourse-io-be/internal/instructor/domain"
)

func TestValidateCertificatePayloadRejectsPartialRow(t *testing.T) {
	t.Parallel()
	err := validateCertificatePayload([]domain.Certificate{
		{Title: "", Issuer: "AWS", IssuedYear: 2024},
	})
	if err == nil {
		t.Fatal("expected error for certificate row with empty title and issuer")
	}
}

func TestValidateCertificatePayloadSkipsEmptyRow(t *testing.T) {
	t.Parallel()
	err := validateCertificatePayload([]domain.Certificate{
		{Title: "", Issuer: "", IssuedYear: 2026},
	})
	if err != nil {
		t.Fatalf("expected empty certificate row to be skipped, got %v", err)
	}
}

func TestCertificateRowHasPartialData(t *testing.T) {
	t.Parallel()
	if !certificateRowHasPartialData(domain.Certificate{Issuer: "X"}) {
		t.Fatal("issuer alone should count as partial data")
	}
	if certificateRowHasPartialData(domain.Certificate{IssuedYear: 2026}) {
		t.Fatal("issued year alone should not count as partial data")
	}
}

func TestValidateCertificatePayloadRejectsDuplicates(t *testing.T) {
	t.Parallel()
	const dupFileID = "11111111-1111-1111-1111-111111111111"
	cases := []struct {
		name  string
		certs []domain.Certificate
	}{
		{
			name: "composite",
			certs: []domain.Certificate{
				{Title: "AWS SA", Issuer: "AWS", IssuedYear: 2024, CredentialURL: "https://example.com/a"},
				{Title: "AWS SA", Issuer: "AWS", IssuedYear: 2024, CredentialURL: "https://example.com/b"},
			},
		},
		{
			name: "credential_url",
			certs: []domain.Certificate{
				{Title: "Cert A", Issuer: "AWS", IssuedYear: 2024, CredentialURL: "https://example.com/same"},
				{Title: "Cert B", Issuer: "GCP", IssuedYear: 2023, CredentialURL: "https://example.com/same"},
			},
		},
		{
			name: "file_id",
			certs: []domain.Certificate{
				{Title: "Cert A", Issuer: "AWS", IssuedYear: 2024, CertificateFileID: dupFileID},
				{Title: "Cert B", Issuer: "GCP", IssuedYear: 2023, CertificateFileID: dupFileID},
			},
		},
		{
			name: "normalized_composite",
			certs: []domain.Certificate{
				{Title: "AWS  Certified", Issuer: "Amazon", IssuedYear: 2024, CredentialURL: "https://example.com/a"},
				{Title: "aws certified", Issuer: "amazon", IssuedYear: 2024, CredentialURL: "https://example.com/b"},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateCertificatePayload(tc.certs)
			if !stderrors.Is(err, domain.ErrDuplicateCertificate) {
				t.Fatalf("expected ErrDuplicateCertificate, got %v", err)
			}
		})
	}
}

func TestValidateCertificatePayloadAcceptsDistinctRows(t *testing.T) {
	t.Parallel()
	err := validateCertificatePayload([]domain.Certificate{
		{Title: "Cert A", Issuer: "AWS", IssuedYear: 2024, CredentialURL: "https://example.com/a"},
		{Title: "Cert B", Issuer: "GCP", IssuedYear: 2023, CertificateFileID: "11111111-1111-1111-1111-111111111111"},
	})
	if err != nil {
		t.Fatalf("expected distinct rows to pass, got %v", err)
	}
}

func assertSubmitProfileFields(t *testing.T, p domain.ProfilePayload, wantErr bool) {
	t.Helper()
	topics := []string{"00000000-0000-0000-0000-000000000010"}
	skills := []string{"00000000-0000-0000-0000-000000000020"}
	err := validateSubmitProfileFields(p, topics, skills)
	if wantErr {
		if !stderrors.Is(err, domain.ErrInvalidApplicationPayload) {
			t.Fatalf("expected ErrInvalidApplicationPayload, got %v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestValidateSubmitProfileFieldsIdeasLengthBoundaries(t *testing.T) {
	t.Parallel()
	base := validSubmitInput().ProfilePayload
	const vietRune = "ý"
	if len(vietRune) == 1 {
		t.Fatal("fixture expects multi-byte Vietnamese rune")
	}

	cases := []struct {
		name    string
		ideas   string
		wantErr bool
	}{
		{name: "ideas_ok", ideas: validTeachingContentIdeas},
		{name: "ideas_trim_whitespace_to_valid", ideas: "  " + strings.Repeat("a", 50) + "  "},
		{name: "ideas_exact_min_50_ascii", ideas: strings.Repeat("a", 50)},
		{name: "ideas_exact_max_500_ascii", ideas: strings.Repeat("a", 500)},
		{name: "ideas_exact_min_50_vietnamese_runes", ideas: strings.Repeat(vietRune, 50)},
		{name: "ideas_exact_max_500_vietnamese_runes", ideas: strings.Repeat(vietRune, 500)},
		{name: "ideas_too_short_49", ideas: strings.Repeat("a", 49), wantErr: true},
		{name: "ideas_too_long_501", ideas: strings.Repeat("a", 501), wantErr: true},
		{name: "ideas_49_vietnamese_runes_too_short", ideas: strings.Repeat(vietRune, 49), wantErr: true},
		{name: "ideas_501_vietnamese_runes_too_long", ideas: strings.Repeat(vietRune, 501), wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := base
			p.TeachingContentIdeas = tc.ideas
			assertSubmitProfileFields(t, p, tc.wantErr)
		})
	}
}

func TestValidateSubmitProfileFieldsBioLengthBoundaries(t *testing.T) {
	t.Parallel()
	base := validSubmitInput().ProfilePayload
	const vietRune = "ý"
	if len(vietRune) == 1 {
		t.Fatal("fixture expects multi-byte Vietnamese rune")
	}

	deltaBio := func(text string) string {
		return `{"ops":[{"insert":` + strconv.Quote(text) + `}]}`
	}

	cases := []struct {
		name    string
		bio     string
		wantErr bool
	}{
		{name: "bio_exact_min_100_vietnamese_runes", bio: strings.Repeat(vietRune, 100)},
		{name: "bio_exact_max_2000_vietnamese_runes", bio: strings.Repeat(vietRune, 2000)},
		{name: "bio_trim_whitespace_to_valid", bio: "  " + strings.Repeat("a", 100) + "  "},
		{name: "bio_delta_exact_min_100", bio: deltaBio(strings.Repeat("a", 100))},
		{name: "bio_delta_exact_max_2000", bio: deltaBio(strings.Repeat("a", 2000))},
		{name: "bio_delta_includes_whitespace_runes", bio: deltaBio(strings.Repeat("a", 99) + " ")},
		{name: "bio_too_short_99", bio: strings.Repeat("a", 99), wantErr: true},
		{name: "bio_too_long_2001", bio: strings.Repeat("a", 2001), wantErr: true},
		{name: "bio_99_vietnamese_runes_too_short", bio: strings.Repeat(vietRune, 99), wantErr: true},
		{name: "bio_delta_too_short_99", bio: deltaBio(strings.Repeat("a", 99)), wantErr: true},
		{name: "bio_delta_too_long_2001", bio: deltaBio(strings.Repeat("a", 2001)), wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := base
			p.Bio = tc.bio
			assertSubmitProfileFields(t, p, tc.wantErr)
		})
	}
}

func TestValidateBioDelta(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		bio     string
		wantErr bool
	}{
		{name: "empty_ok", bio: ""},
		{name: "legacy_plain_ok", bio: "just plain text, not JSON"},
		{name: "json_without_ops_is_legacy_plain_ok", bio: `{"a":1}`},
		{name: "json_ops_null_is_legacy_plain_ok", bio: `{"ops":null}`},
		{name: "delta_plain_insert_ok", bio: `{"ops":[{"insert":"hello\n"}]}`},
		{name: "delta_allowed_attrs_ok", bio: `{"ops":[{"insert":"hi","attributes":{"bold":true,"italic":true,"underline":true,"strike":true}},{"insert":"\n","attributes":{"list":"ordered"}}]}`},
		{name: "delta_image_embed_rejected", bio: `{"ops":[{"insert":{"image":"https://cdn.example.com/x.png"}},{"insert":"text"}]}`, wantErr: true},
		{name: "delta_video_embed_rejected", bio: `{"ops":[{"insert":{"video":"https://cdn.example.com/x.mp4"}}]}`, wantErr: true},
		{name: "delta_document_embed_rejected", bio: `{"ops":[{"insert":{"document":"https://cdn.example.com/x.pdf"}}]}`, wantErr: true},
		{name: "delta_font_attr_rejected", bio: `{"ops":[{"insert":"hi","attributes":{"font":"serif"}}]}`, wantErr: true},
		{name: "delta_size_attr_rejected", bio: `{"ops":[{"insert":"hi","attributes":{"size":"huge"}}]}`, wantErr: true},
		{name: "delta_header_attr_rejected", bio: `{"ops":[{"insert":"\n","attributes":{"header":1}}]}`, wantErr: true},
		{name: "delta_unknown_attr_rejected", bio: `{"ops":[{"insert":"hi","attributes":{"color":"#f00"}}]}`, wantErr: true},
		{name: "delta_non_object_op_rejected", bio: `{"ops":[42]}`, wantErr: true},
		{name: "oversized_raw_rejected", bio: strings.Repeat("a", 32*1024+1), wantErr: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateBioDelta(tc.bio)
			if tc.wantErr {
				if !stderrors.Is(err, domain.ErrInvalidApplicationPayload) {
					t.Fatalf("expected ErrInvalidApplicationPayload, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected ok, got %v", err)
			}
		})
	}
}

// bioDeltaWithEmbedAndValidLength is a bypass payload: an image embed plus
// enough text to satisfy the 100–2000 length rule. Structural validation must
// reject it on both write paths (finding: API could bypass FE-only rules).
func bioDeltaWithEmbedAndValidLength() string {
	return `{"ops":[{"insert":{"image":"https://cdn.example.com/x.png"}},{"insert":` +
		strconv.Quote(strings.Repeat("a", 150)) + `}]}`
}

func TestSubmitApplicationRejectsBioDeltaPolicyBypass(t *testing.T) {
	t.Parallel()
	svc := newAppTestService(&appTestRepo{}, appTestPerms{}, appTestRoles{})
	in := validSubmitInput()
	in.Bio = bioDeltaWithEmbedAndValidLength()
	_, err := svc.SubmitApplication(context.Background(), in, "")
	if !stderrors.Is(err, domain.ErrInvalidApplicationPayload) {
		t.Fatalf("expected ErrInvalidApplicationPayload, got %v", err)
	}
}

func TestUpsertProfileRejectsBioDeltaPolicyBypass(t *testing.T) {
	t.Parallel()
	svc := newAppTestService(&appTestRepo{}, appTestPerms{}, appTestRoles{})
	in := domain.UpsertProfileInput{UserID: "user-1"}
	in.Bio = bioDeltaWithEmbedAndValidLength()
	_, err := svc.UpsertProfile(context.Background(), in)
	if !stderrors.Is(err, domain.ErrInvalidApplicationPayload) {
		t.Fatalf("expected ErrInvalidApplicationPayload, got %v", err)
	}
}
