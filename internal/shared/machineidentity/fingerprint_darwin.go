//go:build darwin

package machineidentity

import (
	"fmt"
	"os/exec"
	"strings"
)

func readPlatformMachineIDs() (machineID, hardwareUUID string, err error) {
	uuid, err := darwinIOPlatformUUID()
	if err != nil {
		return "", "", err
	}
	return uuid, uuid, nil
}

func darwinIOPlatformUUID() (string, error) {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", fmt.Errorf("ioreg IOPlatformUUID: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "IOPlatformUUID") {
			continue
		}
		parts := strings.Split(line, "\"")
		if len(parts) >= 4 {
			uuid := strings.TrimSpace(parts[3])
			if uuid != "" {
				return uuid, nil
			}
		}
	}
	return "", fmt.Errorf("IOPlatformUUID not found")
}
