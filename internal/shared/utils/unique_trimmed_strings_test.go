package utils_test

import (
	"errors"
	"testing"

	"mycourse-io-be/internal/shared/utils"
)

func TestValidateUniqueTrimmedStrings(t *testing.T) {
	t.Parallel()
	emptyErr := errors.New("empty")
	dupErr := errors.New("dup")

	got, err := utils.ValidateUniqueTrimmedStrings([]string{" a ", "b", "a"}, emptyErr, dupErr)
	if err != dupErr {
		t.Fatalf("expected dup err, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil slice on dup, got %v", got)
	}

	got, err = utils.ValidateUniqueTrimmedStrings([]string{"key1", " key2 "}, emptyErr, dupErr)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 2 || got[0] != "key1" || got[1] != "key2" {
		t.Fatalf("got %v", got)
	}

	_, err = utils.ValidateUniqueTrimmedStrings([]string{"  "}, emptyErr, dupErr)
	if err != emptyErr {
		t.Fatalf("expected empty err, got %v", err)
	}
}
