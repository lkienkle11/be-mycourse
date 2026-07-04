package application

import (
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
