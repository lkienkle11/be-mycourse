package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"mycourse-io-be/internal/shared/xdgx"
)

// PathConfig controls default log directory resolution when LOG_DIR is not set.
type PathConfig struct {
	AppName  string
	Vendor   string
	PathMode string // user | service
}

func resolvePathMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "service") {
		return "service"
	}
	return "user"
}

func safePathName(segment string) string {
	s := strings.TrimSpace(segment)
	s = strings.ReplaceAll(s, string(filepath.Separator), "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return s
}

// ResolveLogDir resolves directory for rotated logs.
// LOG_DIR env/setting override always wins and supports os.ExpandEnv.
func ResolveLogDir(cfg PathConfig, explicit string) (string, error) {
	if v := strings.TrimSpace(explicit); v != "" {
		return filepath.Clean(os.ExpandEnv(v)), nil
	}

	app := safePathName(cfg.AppName)
	if app == "" {
		return "", fmt.Errorf("logger: app name is required")
	}
	vendor := safePathName(cfg.Vendor)
	mode := resolvePathMode(cfg.PathMode)

	switch runtime.GOOS {
	case "windows":
		return resolveWindowsLogDir(mode, vendor, app)
	case "darwin":
		return resolveDarwinLogDir(mode, vendor, app)
	default:
		return resolveUnixLogDir(mode, app)
	}
}

func resolveWindowsLogDir(mode, vendor, app string) (string, error) {
	if mode == "service" {
		base := strings.TrimSpace(os.Getenv("ProgramData"))
		if base == "" {
			base = `C:\ProgramData`
		}
		if vendor == "" {
			return filepath.Join(base, app, "logs"), nil
		}
		return filepath.Join(base, vendor, app, "logs"), nil
	}

	base := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if base == "" {
		home, err := xdgx.UserHome()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	if vendor == "" {
		return filepath.Join(base, app, "logs"), nil
	}
	return filepath.Join(base, vendor, app, "logs"), nil
}

func resolveDarwinLogDir(mode, vendor, app string) (string, error) {
	if mode == "service" {
		if vendor == "" {
			return filepath.Join("/Library/Logs", app), nil
		}
		return filepath.Join("/Library/Logs", vendor, app), nil
	}
	home, err := xdgx.UserHome()
	if err != nil {
		return "", err
	}
	if vendor == "" {
		return filepath.Join(home, "Library", "Logs", app), nil
	}
	return filepath.Join(home, "Library", "Logs", vendor, app), nil
}

func resolveUnixLogDir(mode, app string) (string, error) {
	if mode == "service" {
		return filepath.Join("/var/log", app), nil
	}
	base, err := xdgx.StateHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, app, "logs"), nil
}
