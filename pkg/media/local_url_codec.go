package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

func EncodeLocalObjectKey(secret, objectKey string) string {
	msg := strings.TrimSpace(objectKey)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	payload := base64.RawURLEncoding.EncodeToString([]byte(msg))
	return payload + "." + sig
}

func DecodeLocalObjectKey(secret, token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid token payload")
	}
	expected := EncodeLocalObjectKey(secret, string(payload))
	if expected != token {
		return "", errors.New("invalid token signature")
	}
	return string(payload), nil
}

func DecodeLocalURLToken(token string) (string, error) {
	secret := strings.TrimSpace(os.Getenv("LOCAL_FILE_URL_SECRET"))
	if secret == "" {
		secret = "mycourse-local-file-secret"
	}
	objectKey, err := DecodeLocalObjectKey(secret, token)
	if err != nil {
		return "", errors.New("invalid local media token")
	}
	return objectKey, nil
}
