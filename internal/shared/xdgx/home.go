package xdgx

import (
	"os"
	"path/filepath"
	"strings"
)

// UserHome returns the current user's home directory.
func UserHome() (string, error) {
	return os.UserHomeDir()
}

// ConfigHome resolves XDG config home: $XDG_CONFIG_HOME or ~/.config.
func ConfigHome() (string, error) {
	if v := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// StateHome resolves XDG state home: $XDG_STATE_HOME or ~/.local/state.
func StateHome() (string, error) {
	if v := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}
