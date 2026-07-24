package application

import (
	stderrors "errors"
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

	cases := []struct {
		name    string
		bio     string
		wantErr bool
	}{
		{name: "bio_exact_min_100_vietnamese_runes", bio: strings.Repeat(vietRune, 100)},
		{name: "bio_exact_max_2000_vietnamese_runes", bio: strings.Repeat(vietRune, 2000)},
		{name: "bio_trim_whitespace_to_valid", bio: "  " + strings.Repeat("a", 100) + "  "},
		{name: "bio_too_short_99", bio: strings.Repeat("a", 99), wantErr: true},
		{name: "bio_too_long_2001", bio: strings.Repeat("a", 2001), wantErr: true},
		{name: "bio_99_vietnamese_runes_too_short", bio: strings.Repeat(vietRune, 99), wantErr: true},
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
