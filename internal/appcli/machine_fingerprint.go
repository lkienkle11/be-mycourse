package appcli

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const hybridBindingVersion = "v1"

type machineFingerprint struct {
	MachineID    string
	HardwareUUID string
	Hostname     string
	Platform     string
}

// buildHybridMachineBindingMaterial combines enrollment file secret with live OS fingerprint.
// HMAC input = versioned canonical string; DeriveMachineSecret hashes this with app_system_env.
func buildHybridMachineBindingMaterial(fileSecret string) (string, error) {
	fileSecret = strings.TrimSpace(fileSecret)
	if fileSecret == "" {
		return "", fmt.Errorf("machine file secret is empty")
	}
	fp, err := collectLocalMachineFingerprint()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(fp.MachineID) == "" && strings.TrimSpace(fp.HardwareUUID) == "" {
		return "", fmt.Errorf("could not read local machine fingerprint")
	}
	return buildHybridMaterialFromParts(fileSecret, fp), nil
}

func buildHybridMaterialFromParts(fileSecret string, fp machineFingerprint) string {
	// Fixed field order — changing order or version invalidates existing bindings (re-register required).
	return strings.Join([]string{
		hybridBindingVersion,
		"file:" + fileSecret,
		"mid:" + strings.TrimSpace(fp.MachineID),
		"hw:" + strings.TrimSpace(fp.HardwareUUID),
		"host:" + strings.TrimSpace(fp.Hostname),
		"plat:" + strings.TrimSpace(fp.Platform),
	}, "|")
}

func collectLocalMachineFingerprint() (machineFingerprint, error) {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	mid, hw, err := readPlatformMachineIDs()
	if err != nil {
		return machineFingerprint{}, err
	}
	return machineFingerprint{
		MachineID:    mid,
		HardwareUUID: hw,
		Hostname:     host,
		Platform:     runtime.GOOS + "/" + runtime.GOARCH,
	}, nil
}
