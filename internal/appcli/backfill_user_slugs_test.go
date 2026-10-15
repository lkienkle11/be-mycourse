package appcli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mycourse-io-be/internal/auth/infra"
)

type fakeSlugBackfiller struct {
	updated   int
	err       error
	gotBatch  int
	callCount int
}

func (f *fakeSlugBackfiller) BackfillMissingSlugs(_ context.Context, batchSize int) (int, error) {
	f.callCount++
	f.gotBatch = batchSize
	return f.updated, f.err
}

func tempOutputs(t *testing.T) (stdout, stderr *os.File, read func(*os.File) string) {
	t.Helper()
	dir := t.TempDir()
	open := func(name string) *os.File {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	read = func(f *os.File) string {
		b, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	return open("stdout"), open("stderr"), read
}

func TestRunBackfillUserSlugsReportsUpdatedCount(t *testing.T) {
	stdout, stderr, read := tempOutputs(t)
	repo := &fakeSlugBackfiller{updated: 42}

	runBackfillUserSlugs(context.Background(), repo, stdout, stderr)

	if repo.callCount != 1 || repo.gotBatch != infra.DefaultSlugBackfillBatchSize {
		t.Fatalf("calls=%d batch=%d, want 1 call with default batch size", repo.callCount, repo.gotBatch)
	}
	if out := read(stdout); !strings.Contains(out, "updated_users=42") {
		t.Fatalf("stdout = %q, want updated_users=42", out)
	}
	if read(stderr) != "" {
		t.Fatalf("stderr must be empty on success, got %q", read(stderr))
	}
}

func TestRunBackfillUserSlugsReportsFailureWithPartialCount(t *testing.T) {
	stdout, stderr, read := tempOutputs(t)
	repo := &fakeSlugBackfiller{updated: 7, err: errors.New("db down")}

	runBackfillUserSlugs(context.Background(), repo, stdout, stderr)

	errOut := read(stderr)
	if !strings.Contains(errOut, "Failure") || !strings.Contains(errOut, "updating 7 users") || !strings.Contains(errOut, "db down") {
		t.Fatalf("stderr = %q, want failure with partial count and cause", errOut)
	}
	if read(stdout) != "" {
		t.Fatalf("stdout must be empty on failure, got %q", read(stdout))
	}
}

func TestMaybeRunBackfillUserSlugsIsGatedByEnv(t *testing.T) {
	t.Setenv("CLI_BACKFILL_USER_SLUGS", "")
	if MaybeRunBackfillUserSlugs(nil) {
		t.Fatal("must not run when CLI_BACKFILL_USER_SLUGS is unset")
	}
}
