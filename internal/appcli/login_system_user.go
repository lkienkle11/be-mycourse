package appcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/parsebool"
	sysinfra "mycourse-io-be/internal/system/infra"
)

// MaybeRunSystemLogin returns true if the process handled CLI login and should exit.
func MaybeRunSystemLogin(db *gorm.DB) bool {
	if !parsebool.EnvEnabled("CLI_SYSTEM_LOGIN") {
		return false
	}
	runLogin(db)
	return true
}

func runLogin(db *gorm.DB) {
	if err := guardCLIOperation(context.Background(), constants.CLIOpSystemLogin); err != nil {
		printCLIGuardFailure(err)
		return
	}
	username, userPw, ok := cliReadSystemUserCredentials()
	if !ok {
		return
	}
	material, err := LoadMachineIdentityMaterial()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: %v\n", err)
		return
	}
	cfgRepo := sysinfra.NewGormAppConfigRepository(db)
	cfg, err := cfgRepo.Get(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not load system configuration (%v).\n", err)
		return
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" || strings.TrimSpace(cfg.AppTokenEnv) == "" {
		fmt.Fprintln(os.Stderr, "Failure: system token secrets are not configured in database.")
		return
	}
	machineSecret := DeriveMachineSecret(cfg.AppSystemEnv, material)
	sysSvc := newSystemService(db)
	tok, err := sysSvc.SystemLogin(context.Background(), username, userPw, machineSecret)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrSystemMachineBindingFailed):
			fmt.Fprintln(os.Stderr, "Failure: system account is bound to another machine.")
		case errors.Is(err, apperrors.ErrSystemLoginFailed):
			fmt.Fprintln(os.Stderr, "Failure: invalid system credentials.")
		case errors.Is(err, apperrors.ErrSystemSecretsNotReady):
			fmt.Fprintln(os.Stderr, "Failure: system token secrets are not configured in database.")
		default:
			fmt.Fprintf(os.Stderr, "Failure: login failed (%v).\n", err)
		}
		return
	}
	fmt.Println(tok)
}
