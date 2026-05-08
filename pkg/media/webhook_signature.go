package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/setting"
)

// BunnyWebhookSigningSecret returns the signing secret for Bunny webhook validation.
// Source of truth is MediaSetting: read-only key first, then API key fallback for backward compatibility.
func BunnyWebhookSigningSecret() string {
	if key := strings.TrimSpace(setting.MediaSetting.BunnyStreamReadOnlyAPIKey); key != "" {
		return key
	}
	return strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
}

func BunnyWebhookSignatureExpectedHex(rawBody []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}

func IsBunnyWebhookSignatureValid(rawBody []byte, signature, version, algorithm, secret string) bool {
	if strings.TrimSpace(version) != constants.BunnyWebhookSignatureVersionV1 {
		return false
	}
	if strings.TrimSpace(strings.ToLower(algorithm)) != constants.BunnyWebhookSignatureAlgorithmHMAC {
		return false
	}
	received := strings.TrimSpace(strings.ToLower(signature))
	if len(received) != 64 {
		return false
	}
	for _, c := range received {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	expected := BunnyWebhookSignatureExpectedHex(rawBody, secret)
	if len(expected) != len(received) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(received)) == 1
}
