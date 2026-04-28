package helper

import (
	"crypto/sha256"
	"encoding/hex"
)

func ContentFingerprint(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
