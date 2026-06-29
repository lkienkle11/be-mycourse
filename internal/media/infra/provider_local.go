package infra

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

func effectiveLocalFileURLSecret() (string, error) {
	secret := strings.TrimSpace(setting.MediaSetting.LocalFileURLSecret)
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("LOCAL_FILE_URL_SECRET"))
	}
	if secret == "" {
		return "", apperrors.ErrDependencyNotConfigured
	}
	return secret, nil
}

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
		return "", apperrors.ErrLocalMediaTokenInvalidFormat
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", apperrors.ErrLocalMediaTokenInvalidPayload
	}
	expected := EncodeLocalObjectKey(secret, string(payload))
	if expected != token {
		return "", apperrors.ErrLocalMediaTokenInvalidSignature
	}
	return string(payload), nil
}

func DecodeLocalURLToken(token string) (string, error) {
	secret, err := effectiveLocalFileURLSecret()
	if err != nil {
		return "", err
	}
	objectKey, err := DecodeLocalObjectKey(secret, token)
	if err != nil {
		return "", errors.Join(apperrors.ErrLocalMediaTokenInvalid, err)
	}
	return objectKey, nil
}

func UploadLocal(_ *CloudClients, objectKey string, _ domain.RawMetadata) (domain.ProviderUploadResult, error) {
	secret, err := effectiveLocalFileURLSecret()
	if err != nil {
		return domain.ProviderUploadResult{}, err
	}
	token := EncodeLocalObjectKey(secret, objectKey)
	path := "/api/v1/media/files/local/" + token
	return domain.ProviderUploadResult{
		URL:       path,
		OriginURL: objectKey,
		ObjectKey: objectKey,
		Metadata:  domain.RawMetadata{},
	}, nil
}

func BuildPublicURL(provider string, objectKey string) (string, error) {
	switch provider {
	case constants.FileProviderLocal:
		secret, err := effectiveLocalFileURLSecret()
		if err != nil {
			return "", err
		}
		return "/api/v1/media/files/local/" + EncodeLocalObjectKey(secret, objectKey), nil
	case constants.FileProviderBunny:
		base := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
		libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
		if libraryID == "" {
			libraryID = "00000"
		}
		return fmt.Sprintf("%s/%s/%s", base, libraryID, objectKey), nil
	default:
		return buildR2PublicURL(objectKey), nil
	}
}
