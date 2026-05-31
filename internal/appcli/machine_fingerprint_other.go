//go:build !linux && !darwin && !windows

package appcli

import "fmt"

func readPlatformMachineIDs() (machineID, hardwareUUID string, err error) {
	return "", "", fmt.Errorf("unsupported platform for machine fingerprint")
}
