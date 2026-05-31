package appcli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"

	authinfra "mycourse-io-be/internal/auth/infra"
	"mycourse-io-be/internal/shared/parsebool"
	sysinfra "mycourse-io-be/internal/system/infra"
)

// MaybeRunRegisterNewSystemUser returns true if the process handled CLI registration and should exit.
func MaybeRunRegisterNewSystemUser(db *gorm.DB) bool {
	if !parsebool.EnvEnabled("CLI_REGISTER_NEW_SYSTEM_USER") {
		return false
	}
	runRegister(db)
	return true
}

func runRegister(db *gorm.DB) {
	if !cliVerifyAppPassword(db) {
		return
	}
	username, userPw, ok := cliReadSystemUserCredentials()
	if !ok {
		return
	}
	material, err := LoadOrCreateMachineIdentityMaterial()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not load machine identity (%v).\n", err)
		return
	}
	cfgRepo := sysinfra.NewGormAppConfigRepository(db)
	cfg, err := cfgRepo.Get(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not load system configuration (%v).\n", err)
		return
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" {
		fmt.Fprintln(os.Stderr, "Failure: app_system_env is not configured in database (system_app_config).")
		return
	}
	machineSecret := DeriveMachineSecret(cfg.AppSystemEnv, material)
	sysSvc := newSystemService(db)
	if err := sysSvc.RegisterPrivilegedUser(context.Background(), username, userPw, machineSecret); err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not register user (%v).\n", err)
		return
	}
	fmt.Fprintln(os.Stderr, "Success: privileged system user registered.")
}

func cliVerifyAppPassword(db *gorm.DB) bool {
	appPw, err := readSecretInput("Enter app password:")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failure: could not read app password.")
		return false
	}
	cfgRepo := sysinfra.NewGormAppConfigRepository(db)
	cfg, err := cfgRepo.Get(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not load system configuration (%v).\n", err)
		return false
	}
	if strings.TrimSpace(cfg.AppCLISystemPassword) == "" {
		fmt.Fprintln(os.Stderr, "Failure: APP_CLI_SYSTEM_PASSWORD is not set in database (system_app_config).")
		return false
	}
	storedHash := strings.TrimSpace(cfg.AppCLISystemPassword)
	if !authinfra.CheckPassword(appPw, storedHash) {
		fmt.Fprintln(os.Stderr, "Failure: invalid app password.")
		return false
	}
	return true
}
