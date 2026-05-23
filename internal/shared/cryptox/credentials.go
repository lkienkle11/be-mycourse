package cryptox

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CredentialHMACHex returns a deterministic hex digest for credential material
// using secret as HMAC keying material.
func CredentialHMACHex(secret, plain string) []byte {
	k := sha256.Sum256([]byte(secret))
	mac := hmac.New(sha256.New, k[:])
	_, _ = mac.Write([]byte(plain))
	return mac.Sum(nil)
}

// CredentialHMACHEXString is the hex-encoded form of CredentialHMACHex.
func CredentialHMACHEXString(secret, plain string) string {
	return hex.EncodeToString(CredentialHMACHex(secret, plain))
}

// JWTKeyFromEnv derives a 32-byte HS256 key from an environment secret string.
func JWTKeyFromEnv(tokenEnv string) []byte {
	k := sha256.Sum256([]byte(tokenEnv))
	return k[:]
}
