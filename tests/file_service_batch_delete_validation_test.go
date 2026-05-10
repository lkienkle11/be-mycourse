package tests

import (
	"errors"
	"strings"
	"testing"

	"mycourse-io-be/constants"
	pkgerrors "mycourse-io-be/pkg/errors"
	mediaservice "mycourse-io-be/services/media"
)

func TestDeleteFilesByObjectKeys_rejectsMoreThanMaxBatch(t *testing.T) {
	keys := make([]string, constants.MaxMediaBatchDelete+1)
	for i := range keys {
		keys[i] = strings.Repeat("k", i+1)
	}
	err := mediaservice.DeleteFilesByObjectKeys(keys)
	if !errors.Is(err, pkgerrors.ErrMediaBatchDeleteTooManyIDs) {
		t.Fatalf("got %v want ErrMediaBatchDeleteTooManyIDs", err)
	}
}

func TestDeleteFilesByObjectKeys_rejectsDuplicateKeys(t *testing.T) {
	err := mediaservice.DeleteFilesByObjectKeys([]string{"same-object-key", "same-object-key"})
	if !errors.Is(err, pkgerrors.ErrMediaDuplicateObjectKeysInBatchDelete) {
		t.Fatalf("got %v want ErrMediaDuplicateObjectKeysInBatchDelete", err)
	}
}

func TestDeleteFilesByObjectKeys_rejectsEmptyKeyAfterTrim(t *testing.T) {
	err := mediaservice.DeleteFilesByObjectKeys([]string{"valid-looking-key", "   "})
	if !errors.Is(err, pkgerrors.ErrMediaObjectKeyRequired) {
		t.Fatalf("got %v want ErrMediaObjectKeyRequired", err)
	}
}
