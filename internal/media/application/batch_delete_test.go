package application_test

import (
	"errors"
	"strings"
	"testing"

	mediaapp "mycourse-io-be/internal/media/application"
	apperrors "mycourse-io-be/internal/shared/errors"
)

func TestDeleteFilesByObjectKeys_rejectsMoreThanMaxBatch(t *testing.T) {
	keys := make([]string, 201) // MaxMediaBatchDelete is 200
	for i := range keys {
		keys[i] = strings.Repeat("k", i+1)
	}
	err := mediaapp.ValidateBatchDeleteKeys(keys)
	if !errors.Is(err, apperrors.ErrMediaBatchDeleteTooManyIDs) {
		t.Fatalf("got %v want ErrMediaBatchDeleteTooManyIDs", err)
	}
}

func TestDeleteFilesByObjectKeys_rejectsDuplicateKeys(t *testing.T) {
	err := mediaapp.ValidateBatchDeleteKeys([]string{"same-object-key", "same-object-key"})
	if !errors.Is(err, apperrors.ErrMediaDuplicateObjectKeysInBatchDelete) {
		t.Fatalf("got %v want ErrMediaDuplicateObjectKeysInBatchDelete", err)
	}
}

func TestDeleteFilesByObjectKeys_rejectsEmptyKeyAfterTrim(t *testing.T) {
	err := mediaapp.ValidateBatchDeleteKeys([]string{"valid-looking-key", "   "})
	if !errors.Is(err, apperrors.ErrMediaObjectKeyRequired) {
		t.Fatalf("got %v want ErrMediaObjectKeyRequired", err)
	}
}
