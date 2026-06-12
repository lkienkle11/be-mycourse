package machineidentity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mycourse-io-be/internal/shared/xdgx"
)

const machineIdentityDir = "mycourse"
const machineIdentityFile = "machine_identity"

// LoadOrCreateMachineIdentityMaterial returns hybrid binding material:
// enrollment file secret + live OS fingerprint (machine-id, hardware UUID, hostname, platform).
func LoadOrCreateMachineIdentityMaterial() (string, error) {
	secret, err := loadOrCreateFileSecret()
	if err != nil {
		return "", err
	}
	return BuildHybridMachineBindingMaterial(secret)
}

// LoadMachineIdentityMaterial returns hybrid binding material for login (file secret must exist).
func LoadMachineIdentityMaterial() (string, error) {
	secret, err := loadFileSecret()
	if err != nil {
		return "", err
	}
	return BuildHybridMachineBindingMaterial(secret)
}

// IdentityFilePath returns the enrollment secret file path under XDG config home.
func IdentityFilePath() (string, error) {
	return identityFilePath()
}

func loadOrCreateFileSecret() (string, error) {
	path, err := identityFilePath()
	if err != nil {
		return "", err
	}
	if data, err := os.ReadFile(path); err == nil {
		secret := strings.TrimSpace(string(data))
		if secret == "" {
			return "", fmt.Errorf("machine identity file is empty: %s", path)
		}
		return secret, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	secret, err := newFileSecret()
	if err != nil {
		return "", err
	}
	if err := writeMachineIdentityFile(path, secret); err != nil {
		return "", err
	}
	return secret, nil
}

func loadFileSecret() (string, error) {
	path, err := identityFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("machine identity file not found: %s (register a system user on this host first)", path)
		}
		return "", err
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", fmt.Errorf("machine identity file is empty: %s", path)
	}
	return secret, nil
}

func identityFilePath() (string, error) {
	base, err := xdgx.ConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, machineIdentityDir, machineIdentityFile), nil
}

func newFileSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}

func writeMachineIdentityFile(path, secret string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(secret), 0o600)
}
