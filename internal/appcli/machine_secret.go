package appcli

import (
	sysinfra "mycourse-io-be/internal/system/infra"
)

// DeriveMachineSecret returns the machine binding digest for local identity material.
func DeriveMachineSecret(appSystemEnv, material string) string {
	return sysinfra.NewSystemCryptoAdapter().CredentialHMACHex(appSystemEnv, material)
}
