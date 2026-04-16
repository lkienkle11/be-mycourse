package appcli

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
	"gorm.io/gorm"

	"mycourse-io-be/pkg/envbool"
	"mycourse-io-be/services"
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
	appPw, err := readSecretInput("Enter app password:")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failure: could not read app password.")
		return
	}
	cfg, err := services.GetSystemAppConfig(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not load system configuration (%v).\n", err)
		return
	}
	if strings.TrimSpace(cfg.AppCLISystemPassword) == "" {
		fmt.Fprintln(os.Stderr, "Failure: APP_CLI_SYSTEM_PASSWORD is not set in database (system_app_config).")
		return
	}
	want := []byte(strings.TrimSpace(cfg.AppCLISystemPassword))
	got := []byte(strings.TrimSpace(appPw))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		fmt.Fprintln(os.Stderr, "Failure: invalid app password.")
		return
	}

	username, err := readSecretInput("Enter username:")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failure: could not read username.")
		return
	}
	username = strings.TrimSpace(username)

	userPw, err := readSecretInput("Enter password:")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failure: could not read password.")
		return
	}
	if err := services.RegisterSystemPrivilegedUser(db, username, userPw); err != nil {
		fmt.Fprintf(os.Stderr, "Failure: could not register user (%v).\n", err)
		return
	}
	fmt.Fprintln(os.Stderr, "Success: privileged system user registered.")
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
