package appcli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/ratelimit"
	"mycourse-io-be/internal/shared/resilience"
)

// guardCLIOperation checks circuit breaker and file-backed rate limit before credential prompts.
func guardCLIOperation(_ context.Context, opKind string) error {
	if !resilience.Global.Allow() {
		return fmt.Errorf("service temporarily unavailable; try again later")
	}
	key, err := cliRateLimitKey(opKind)
	if err != nil {
		return err
	}
	allowed, err := ratelimit.AllowCLI(key, constants.CLIAttempts, constants.CLIMinutes)
	if err != nil {
		return fmt.Errorf("could not check CLI rate limit (%w)", err)
	}
	if !allowed {
		return fmt.Errorf("too many CLI attempts; try again in a few minutes")
	}
	return nil
}

func cliRateLimitKey(opKind string) (string, error) {
	material, err := stableMachineMaterialForRateLimit()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(material))
	return opKind + "|" + hex.EncodeToString(sum[:8]), nil
}

func stableMachineMaterialForRateLimit() (string, error) {
	path, err := machineIdentityPath()
	if err != nil {
		return fallbackHostPlatformMaterial(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fallbackHostPlatformMaterial(), nil
		}
		return "", err
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return fallbackHostPlatformMaterial(), nil
	}
	return buildHybridMachineBindingMaterial(secret)
}

func fallbackHostPlatformMaterial() string {
	host, _ := os.Hostname()
	return strings.Join([]string{"fallback", host, runtime.GOOS, runtime.GOARCH}, "|")
}

func printCLIGuardFailure(err error) {
	fmt.Fprintf(os.Stderr, "Failure: %s.\n", err.Error())
}
