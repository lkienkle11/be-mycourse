//go:build linux

package machineidentity

import (
	"os"
	"strings"
)

func readPlatformMachineIDs() (machineID, hardwareUUID string, err error) {
	if b, readErr := os.ReadFile("/etc/machine-id"); readErr == nil {
		machineID = strings.TrimSpace(string(b))
	}
	if b, readErr := os.ReadFile("/sys/class/dmi/id/product_uuid"); readErr == nil {
		hardwareUUID = strings.TrimSpace(string(b))
	}
	if machineID == "" {
		machineID = hardwareUUID
	}
	if hardwareUUID == "" {
		hardwareUUID = machineID
	}
	return machineID, hardwareUUID, nil
}
