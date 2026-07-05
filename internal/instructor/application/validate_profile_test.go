package application

import (
	stderrors "errors"
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
