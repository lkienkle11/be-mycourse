package infra

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"mycourse-io-be/internal/instructor/domain"
)

type certificateJSON struct {
	Title         string `json:"title"`
	Issuer        string `json:"issuer"`
	IssuedYear    int    `json:"issued_year"`
	CredentialURL string `json:"credential_url,omitempty"`
}

func certificatesToJSON(certs []domain.Certificate) ([]certificateJSON, error) {
	out := make([]certificateJSON, len(certs))
	for i, c := range certs {
		out[i] = certificateJSON{
			Title: c.Title, Issuer: c.Issuer, IssuedYear: c.IssuedYear, CredentialURL: c.CredentialURL,
		}
	}
	return out, nil
}

func certificatesFromJSON(raw []certificateJSON) []domain.Certificate {
	out := make([]domain.Certificate, len(raw))
	for i, c := range raw {
		out[i] = domain.Certificate{
			Title: c.Title, Issuer: c.Issuer, IssuedYear: c.IssuedYear, CredentialURL: c.CredentialURL,
		}
	}
	return out
}

// StringSliceJSON stores []string in JSONB.
type StringSliceJSON []string

func (s StringSliceJSON) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}

func (s *StringSliceJSON) Scan(value any) error {
	return scanJSONValue(value, s, "StringSliceJSON")
}

// CertificatesJSON stores []domain.Certificate in JSONB.
type CertificatesJSON []certificateJSON

func (c CertificatesJSON) Value() (driver.Value, error) {
	if c == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(c)
}

func (c *CertificatesJSON) Scan(value any) error {
	return scanJSONValue(value, c, "CertificatesJSON")
}

type rejectionHistoryJSON struct {
	RejectedAt          int64  `json:"rejected_at"`
	RejectedByUserID    string `json:"rejected_by_user_id"`
	ReviewerDisplayName string `json:"reviewer_display_name"`
	Reason              string `json:"reason"`
}

// RejectionHistoryJSON stores []domain.RejectionRecord in JSONB.
type RejectionHistoryJSON []rejectionHistoryJSON

func (r RejectionHistoryJSON) Value() (driver.Value, error) {
	if r == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(r)
}

func (r *RejectionHistoryJSON) Scan(value any) error {
	return scanJSONValue(value, r, "RejectionHistoryJSON")
}

func scanJSONValue(value any, dest any, label string) error {
	if value == nil {
		switch d := dest.(type) {
		case *StringSliceJSON:
			*d = []string{}
		case *CertificatesJSON:
			*d = []certificateJSON{}
		case *RejectionHistoryJSON:
			*d = RejectionHistoryJSON{}
		}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("%s: unsupported type %T", label, value)
	}
	return json.Unmarshal(b, dest)
}
