package media

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	errfuncmedia "mycourse-io-be/pkg/errors_func/media"
	"mycourse-io-be/pkg/logger"
	"mycourse-io-be/pkg/logic/mapping"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/response"
	mediaservice "mycourse-io-be/services/media"
)

// bunnyWebhookLog returns a logger tagged for grep/filter in staging and production.
func bunnyWebhookLog(c *gin.Context) *zap.Logger {
	return logger.FromContext(c.Request.Context()).With(
		zap.String("component", "bunny_webhook"),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
	)
}

func readBunnyWebhookRawBody(c *gin.Context) ([]byte, bool) {
	log := bunnyWebhookLog(c).With(zap.String("bunny_webhook_stage", "read_raw_body"))
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Warn("bunny webhook: failed to read request body", zap.Error(err))
		response.Fail(c, http.StatusBadRequest, errcode.InvalidJSON, errcode.DefaultMessage(errcode.InvalidJSON), nil)
		return nil, false
	}
	log.Debug("bunny webhook: raw body read", zap.Int("body_bytes", len(rawBody)))
	return rawBody, true
}

func bunnyWebhookSignatureHeaders(c *gin.Context) (sigHex, version, algorithm string) {
	sigHex = strings.TrimSpace(c.GetHeader(constants.BunnyWebhookSignatureHeader))
	version = strings.TrimSpace(c.GetHeader(constants.BunnyWebhookSignatureVersionHeader))
	algorithm = strings.TrimSpace(c.GetHeader(constants.BunnyWebhookSignatureAlgorithmHeader))
	return sigHex, version, algorithm
}

func failBunnyWebhookMissingSigningSecret(c *gin.Context, log *zap.Logger) {
	log.Warn("bunny webhook: signing secret not configured (read-only key and API key both empty)")
	response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
}

func failBunnyWebhookInvalidSignature(c *gin.Context, log *zap.Logger, ver, alg string, sigLen, rawLen int) {
	log.Warn("bunny webhook: signature validation failed",
		zap.String("header_signature_version", ver),
		zap.String("header_signature_algorithm", alg),
		zap.Int("header_signature_hex_len", sigLen),
		zap.Int("raw_body_bytes", rawLen),
	)
	response.Fail(c, http.StatusUnauthorized, errcode.Unauthorized, errcode.DefaultMessage(errcode.Unauthorized), nil)
}

func verifyBunnyWebhookSignature(c *gin.Context, rawBody []byte) bool {
	log := bunnyWebhookLog(c).With(zap.String("bunny_webhook_stage", "verify_signature"))
	signingSecret := pkgmedia.BunnyWebhookSigningSecret()
	sig, ver, alg := bunnyWebhookSignatureHeaders(c)
	log.Debug("bunny webhook: signature headers snapshot",
		zap.Bool("signing_secret_configured", signingSecret != ""),
		zap.String("header_signature_version", ver),
		zap.String("header_signature_algorithm", alg),
		zap.Int("header_signature_hex_len", len(sig)),
	)
	if signingSecret == "" {
		failBunnyWebhookMissingSigningSecret(c, log)
		return false
	}
	if !pkgmedia.IsBunnyWebhookSignatureValid(
		rawBody,
		c.GetHeader(constants.BunnyWebhookSignatureHeader),
		c.GetHeader(constants.BunnyWebhookSignatureVersionHeader),
		c.GetHeader(constants.BunnyWebhookSignatureAlgorithmHeader),
		signingSecret,
	) {
		failBunnyWebhookInvalidSignature(c, log, ver, alg, len(sig), len(rawBody))
		return false
	}
	log.Debug("bunny webhook: signature ok")
	return true
}

func parseBunnyWebhookRequest(c *gin.Context, log *zap.Logger, rawBody []byte) (dto.BunnyVideoWebhookRequest, bool) {
	req, err := mapping.UnmarshalBunnyVideoWebhookRequestJSON(rawBody)
	if err != nil {
		log.Warn("bunny webhook: JSON unmarshal failed",
			zap.String("bunny_webhook_stage", "json_unmarshal"),
			zap.Error(err),
			zap.Int("raw_body_bytes", len(rawBody)),
		)
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), nil)
		return dto.BunnyVideoWebhookRequest{}, false
	}
	log.Debug("bunny webhook: JSON parsed",
		zap.String("bunny_webhook_stage", "json_unmarshal"),
		zap.Int("video_library_id", req.VideoLibraryID),
		zap.String("video_guid", req.VideoGUID),
		zap.Int("status", req.Status),
	)
	if err := mapping.ValidateBunnyVideoWebhookRequest(req); err != nil {
		log.Warn("bunny webhook: payload validation failed",
			zap.String("bunny_webhook_stage", "validate_payload"),
			zap.Error(err),
			zap.Int("video_library_id", req.VideoLibraryID),
			zap.String("video_guid", req.VideoGUID),
			zap.Int("status", req.Status),
		)
		response.Fail(c, http.StatusUnprocessableEntity, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), nil)
		return dto.BunnyVideoWebhookRequest{}, false
	}
	return req, true
}

func respondBunnyWebhookServiceError(c *gin.Context, log *zap.Logger, err error) {
	if pe, ok := errfuncmedia.AsProviderError(err); ok {
		log.Warn("bunny webhook: service returned provider error",
			zap.String("bunny_webhook_stage", "service_handle_provider_error"),
			zap.Int("provider_code", pe.Code),
			zap.String("provider_message", pe.Error()),
			zap.Error(err),
		)
		msg := pe.Error()
		response.Fail(c, errfuncmedia.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
		return
	}
	log.Warn("bunny webhook: service returned error",
		zap.String("bunny_webhook_stage", "service_handle_error"),
		zap.Error(err),
	)
	response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
}

func bunnyWebhook(c *gin.Context) {
	log := bunnyWebhookLog(c)
	log.Debug("bunny webhook: handler entered")
	rawBody, ok := readBunnyWebhookRawBody(c)
	if !ok {
		return
	}
	if !verifyBunnyWebhookSignature(c, rawBody) {
		return
	}
	req, ok := parseBunnyWebhookRequest(c, log, rawBody)
	if !ok {
		return
	}
	log.Debug("bunny webhook: calling HandleBunnyVideoWebhook",
		zap.String("bunny_webhook_stage", "service_handle_start"),
		zap.Int("video_library_id", req.VideoLibraryID),
		zap.String("video_guid", req.VideoGUID),
		zap.Int("status", req.Status),
	)
	if err := mediaservice.HandleBunnyVideoWebhook(c.Request.Context(), req); err != nil {
		respondBunnyWebhookServiceError(c, log, err)
		return
	}
	log.Debug("bunny webhook: handler completed ok",
		zap.String("bunny_webhook_stage", "service_handle_ok"),
		zap.Int("video_library_id", req.VideoLibraryID),
		zap.String("video_guid", req.VideoGUID),
		zap.Int("status", req.Status),
	)
	response.OK(c, "ok", nil)
}
