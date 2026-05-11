package tests

// Sub 19 — pkg/logger smoke tests (Zap init, context fields, JSON file tee).
// Convention: all tests under repository root `tests/` (docs/patterns.md).

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"mycourse-io-be/pkg/logger"
)

func resetGlobalLogger(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_ = zap.L().Sync()
		zap.ReplaceGlobals(zap.NewNop())
	})
}

func TestFromContext_addsRequestID(t *testing.T) {
	resetGlobalLogger(t)
	core, observed := observer.New(zap.InfoLevel)
	zap.ReplaceGlobals(zap.New(core))

	ctx := logger.WithRequestID(context.Background(), "req-test-1")
	logger.FromContext(ctx).Info("hello")

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	var saw bool
	for _, f := range entries[0].Context {
		if f.Key == "request_id" && f.String == "req-test-1" {
			saw = true
			break
		}
	}
	if !saw {
		t.Fatalf("expected request_id field in log context: %#v", entries[0].Context)
	}
}

func readOneJSONLogLine(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		t.Fatalf("log file line must be JSON: %v\n%s", err, raw)
	}
	return row
}

func TestInit_JSONFile_containsStructuredFields(t *testing.T) {
	resetGlobalLogger(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sub19.log")

	_, err := logger.Init(logger.Options{
		Level:       "info",
		Format:      "json",
		FilePath:    path,
		ServiceName: "test-service",
		Environment: "test",
		Version:     "9.9.9-test",
	})
	if err != nil {
		t.Fatal(err)
	}

	zap.L().Info("marker_event", zap.String("marker_key", "marker_val"))
	logger.Sync()

	row := readOneJSONLogLine(t, path)
	if row["msg"] != "marker_event" {
		t.Fatalf("expected msg marker_event, got %v", row["msg"])
	}
	if row["service"] != "test-service" || row["env"] != "test" || row["version"] != "9.9.9-test" {
		t.Fatalf("expected global fields on file line, got %#v", row)
	}
}

func TestInit_levelError_dropsInfo(t *testing.T) {
	resetGlobalLogger(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "level.log")
	_, err := logger.Init(logger.Options{
		Level:    "error",
		Format:   "json",
		FilePath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	zap.L().Info("should_not_appear")
	zap.L().Error("should_appear")
	logger.Sync()

	row := readOneJSONLogLine(t, path)
	if row["msg"] != "should_appear" {
		t.Fatalf("expected only error-level msg on file sink, got msg=%v", row["msg"])
	}
}
