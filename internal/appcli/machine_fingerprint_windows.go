//go:build windows

package appcli

import (
	"fmt"
	"os/exec"
	"strings"
)

func readPlatformMachineIDs() (machineID, hardwareUUID string, err error) {
	guid, err := windowsMachineGuid()
	if err != nil {
		return "", "", err
	}
	return guid, guid, nil
}

func windowsMachineGuid() (string, error) {
	out, err := exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid").Output()
	if err != nil {
		return "", fmt.Errorf("registry MachineGuid: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(strings.ToLower(line), "machineguid") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			guid := strings.TrimSpace(fields[len(fields)-1])
			if guid != "" {
				return guid, nil
			}
		}
	}
	return "", fmt.Errorf("MachineGuid not found in registry output")
}
