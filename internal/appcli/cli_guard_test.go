package appcli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/resilience"
)

func TestGuardCLIOperationRateLimit(t *testing.T) {
	configHome := t.TempDir()
	identityDir := filepath.Join(configHome, "mycourse")
	if err := os.MkdirAll(identityDir, 0o700); err != nil {
		t.Fatal(err)
	}
	identityPath := filepath.Join(identityDir, "machine_identity")
	if err := os.WriteFile(identityPath, []byte("test-secret-value"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("MYCOURSE_CLI_RATE_LIMIT_PATH", filepath.Join(identityDir, "cli_rate_limit.json"))

	resilience.Global = resilience.NewCircuitBreaker(resilience.DefaultConfig())

	for i := 0; i < constants.CLIAttempts; i++ {
		if err := guardCLIOperation(context.Background(), constants.CLIOpSystemLogin); err != nil {
			t.Fatalf("attempt %d should pass guard: %v", i+1, err)
		}
	}
	if err := guardCLIOperation(context.Background(), constants.CLIOpSystemLogin); err == nil {
		t.Fatal("expected rate limit error on sixth attempt")
	}
}

func TestGuardCLIOperationCircuitOpen(t *testing.T) {
	t.Setenv("MYCOURSE_CLI_RATE_LIMIT_PATH", filepath.Join(t.TempDir(), "cli_rate_limit.json"))
	cb := resilience.NewCircuitBreaker(resilience.DefaultConfig())
	resilience.ForceOpenForTest(cb)
	resilience.Global = cb

	if err := guardCLIOperation(context.Background(), constants.CLIOpRegister); err == nil {
		t.Fatal("expected circuit breaker denial")
	}
}
