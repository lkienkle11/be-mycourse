package appcli

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
	"gorm.io/gorm"

	"mycourse-io-be/internal/system/application"
	sysinfra "mycourse-io-be/internal/system/infra"
)

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
		sysinfra.NewSystemCryptoAdapter(),
	)
}

func cliReadSystemUserCredentials() (username, userPw string, ok bool) {
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
