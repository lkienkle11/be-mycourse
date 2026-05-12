package appcli

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
	"gorm.io/gorm"

	"mycourse-io-be/internal/system/application"
	sysinfra "mycourse-io-be/internal/system/infra"
	"mycourse-io-be/pkg/envbool"
)

// MaybeRunRegisterNewSystemUser returns true if the process handled CLI registration and should exit.
func MaybeRunRegisterNewSystemUser(db *gorm.DB) bool {
	if !envbool.Enabled("CLI_REGISTER_NEW_SYSTEM_USER") {
		return false
	}
	runRegister(db)
	return true
}

func runRegister(db *gorm.DB) {
	if !cliVerifyAppPassword(db) {
		return
	}
	username, userPw, ok := cliReadNewSystemUserCredentials()
	if !ok {
		return
	}
	sysSvc := newSystemService(db)
	if err := sysSvc.RegisterPrivilegedUser(context.Background(), username, userPw); err != nil {
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
	want := []byte(strings.TrimSpace(cfg.AppCLISystemPassword))
	got := []byte(strings.TrimSpace(appPw))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		fmt.Fprintln(os.Stderr, "Failure: invalid app password.")
		return false
	}
	return true
}

func cliReadNewSystemUserCredentials() (username, userPw string, ok bool) {
	u, err := readSecretInput("Enter username:")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failure: could not read username.")
		return "", "", false
	}
	pw, err := readSecretInput("Enter password:")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failure: could not read password.")
		return "", "", false
	}
	return strings.TrimSpace(u), pw, true
}

// readSecretInput prints prompt to stderr, reads a line with local echo disabled (no characters shown).
// If stdin is not a TTY, it tries the controlling console (/dev/tty or CONIN$) so hidden input still works when stdin is redirected.
func readSecretInput(prompt string) (string, error) {
	fmt.Fprintln(os.Stderr, prompt)
	s, err := readLineNoEcho()
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(s), nil
}

func readLineNoEcho() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	tty, err := openControllingConsole()
	if err != nil {
		return "", fmt.Errorf("%w: run this in an interactive terminal (PowerShell, cmd, Windows Terminal)", err)
	}
	defer func() { _ = tty.Close() }()
	tfd := int(tty.Fd())
	if !term.IsTerminal(tfd) {
		return "", errors.New("console device is not a terminal")
	}
	b, err := term.ReadPassword(tfd)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func openControllingConsole() (*os.File, error) {
	if runtime.GOOS == "windows" {
		f, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			f, err = os.OpenFile(`\\.\CONIN$`, os.O_RDWR, 0)
		}
		return f, err
	}
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

func newSystemService(db *gorm.DB) *application.SystemService {
	return application.NewSystemService(
		sysinfra.NewGormAppConfigRepository(db),
		sysinfra.NewGormPrivilegedUserRepository(db),
		sysinfra.NewGormPermissionSyncer(db),
		sysinfra.NewGormRolePermissionSyncer(db),
	)
}
