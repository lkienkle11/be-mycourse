package infra

import "mycourse-io-be/internal/system/domain"

// SystemCryptoAdapter implements domain.SystemCrypto using package-level crypto helpers.
type SystemCryptoAdapter struct{}

// NewSystemCryptoAdapter constructs the default SystemCrypto implementation.
func NewSystemCryptoAdapter() domain.SystemCrypto {
	return SystemCryptoAdapter{}
}

func (SystemCryptoAdapter) CredentialHMACHex(secret, plain string) string {
	return CredentialHMACHex(secret, plain)
}

func (SystemCryptoAdapter) MintSystemAccessToken(tokenEnv, usernameSecretHex string) (string, error) {
	return MintSystemAccessToken(tokenEnv, usernameSecretHex)
}

func (SystemCryptoAdapter) ParseSystemAccessToken(tokenEnv, tokenStr string) (string, error) {
	return ParseSystemAccessToken(tokenEnv, tokenStr)
}
