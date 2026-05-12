package delivery

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mycourse-io-be/internal/media/application"
	mediadomain "mycourse-io-be/internal/media/domain" //nolint:depguard // delivery reads domain webhook/signature constants; no business logic
	mediainfra "mycourse-io-be/internal/media/infra" //nolint:depguard // delivery calls infra.BunnyWebhookSigningSecret config helper; TODO: expose via service interface
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/response"
)

func (h *Handler) bunnyWebhookLog(c *gin.Context) *zap.Logger {
	return logger.FromContext(c.Request.Context()).With(
		zap.String("component", "bunny_webhook"),
		zap.String("path", c.Request.URL.Path),
	)
}

func (h *Handler) readBunnyWebhookRawBody(c *gin.Context) ([]byte, bool) {
	log := h.bunnyWebhookLog(c).With(zap.String("bunny_webhook_stage", "read_raw_body"))
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Warn("bunny webhook: failed to read request body", zap.Error(err))
		response.Fail(c, http.StatusBadRequest, apperrors.InvalidJSON, apperrors.DefaultMessage(apperrors.InvalidJSON), nil)
		return nil, false
	}
	return rawBody, true
}

func (h *Handler) verifyBunnyWebhookSignature(c *gin.Context, rawBody []byte) bool {
	log := h.bunnyWebhookLog(c).With(zap.String("bunny_webhook_stage", "verify_signature"))
	signingSecret := mediainfra.BunnyWebhookSigningSecret()
	sig := strings.TrimSpace(c.GetHeader(mediadomain.BunnyWebhookSignatureHeader))
	ver := strings.TrimSpace(c.GetHeader(mediadomain.BunnyWebhookSignatureVersionHeader))
	alg := strings.TrimSpace(c.GetHeader(mediadomain.BunnyWebhookSignatureAlgorithmHeader))

	if signingSecret == "" {
		log.Warn("bunny webhook: signing secret not configured")
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return false
	}
	if !mediainfra.IsBunnyWebhookSignatureValid(rawBody, sig, ver, alg, signingSecret) {
		log.Warn("bunny webhook: signature validation failed",
			zap.String("header_signature_version", ver),
			zap.String("header_signature_algorithm", alg),
			zap.Int("header_signature_hex_len", len(sig)),
			zap.Int("raw_body_bytes", len(rawBody)),
		)
		response.Fail(c, http.StatusUnauthorized, apperrors.Unauthorized, apperrors.DefaultMessage(apperrors.Unauthorized), nil)
		return false
	}
	return true
}

func (h *Handler) parseBunnyWebhookRequest(c *gin.Context, log *zap.Logger, rawBody []byte) (BunnyVideoWebhookRequest, bool) {
	req, err := unmarshalBunnyWebhookRequest(rawBody)
	if err != nil {
		log.Warn("bunny webhook: JSON unmarshal failed",
			zap.String("bunny_webhook_stage", "json_unmarshal"),
			zap.Error(err), zap.Int("raw_body_bytes", len(rawBody)),
		)
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, apperrors.DefaultMessage(apperrors.ValidationFailed), nil)
		return BunnyVideoWebhookRequest{}, false
	}
	if err := validateBunnyWebhookRequest(req); err != nil {
		log.Warn("bunny webhook: payload validation failed",
			zap.String("bunny_webhook_stage", "validate_payload"),
			zap.Error(err),
		)
		response.Fail(c, http.StatusUnprocessableEntity, apperrors.ValidationFailed, apperrors.DefaultMessage(apperrors.ValidationFailed), nil)
		return BunnyVideoWebhookRequest{}, false
	}
	return req, true
}

func (h *Handler) bunnyWebhook(c *gin.Context) {
	log := h.bunnyWebhookLog(c)
	rawBody, ok := h.readBunnyWebhookRawBody(c)
	if !ok {
		return
	}
	if !h.verifyBunnyWebhookSignature(c, rawBody) {
		return
	}
	req, ok := h.parseBunnyWebhookRequest(c, log, rawBody)
	if !ok {
		return
	}
	input := application.BunnyWebhookInput{VideoGUID: req.VideoGUID, Status: req.Status}
	if err := h.svc.HandleBunnyVideoWebhook(c.Request.Context(), input); err != nil {
		if pe, ok := asProviderError(err); ok {
			msg := pe.Error()
			response.Fail(c, httpStatusForProviderCode(pe.Code), pe.Code, msg, nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", nil)
}
