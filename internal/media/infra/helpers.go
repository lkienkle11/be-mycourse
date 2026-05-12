package infra

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"mycourse-io-be/internal/media/domain"
)

func newUUID() string { return uuid.NewString() }

func strMetaVal(meta domain.RawMetadata, key string) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}
