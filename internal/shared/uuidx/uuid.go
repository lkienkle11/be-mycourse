package uuidx

import (
	"crypto/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// NewV4 returns a random UUID (v4) as string.
func NewV4() string {
	return uuid.NewString()
}

// NewV7 returns a time-ordered UUID (v7) as string.
func NewV7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// NewULID returns a lexicographically sortable ULID string.
func NewULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

// IsValid reports whether s is a valid UUID string.
func IsValid(s string) bool {
	_, err := uuid.Parse(strings.TrimSpace(s))
	return err == nil
}
