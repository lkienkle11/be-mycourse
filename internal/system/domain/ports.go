package domain

// SystemCrypto hashes credentials and mints or verifies short-lived system JWTs.
type SystemCrypto interface {
	CredentialHMACHex(secret, plain string) string
	MintSystemAccessToken(tokenEnv, usernameSecretHex string) (string, error)
	ParseSystemAccessToken(tokenEnv, tokenStr string) (usernameSecretHex string, err error)
}
