package media

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/errcode"
	errfuncmedia "mycourse-io-be/pkg/errors_func/media"
	"mycourse-io-be/pkg/logic/mapping"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/response"
	mediaservice "mycourse-io-be/services/media"
)

func readBunnyWebhookRawBody(c *gin.Context) ([]byte, bool) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.InvalidJSON, errcode.DefaultMessage(errcode.InvalidJSON), nil)
		return nil, false
	}
	return rawBody, true
}

func verifyBunnyWebhookSignature(c *gin.Context, rawBody []byte) bool {
	signingSecret := pkgmedia.BunnyWebhookSigningSecret()
	if signingSecret == "" {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return false
	}
	if !pkgmedia.IsBunnyWebhookSignatureValid(
		rawBody,
		c.GetHeader(constants.BunnyWebhookSignatureHeader),
		c.GetHeader(constants.BunnyWebhookSignatureVersionHeader),
		c.GetHeader(constants.BunnyWebhookSignatureAlgorithmHeader),
		signingSecret,
	) {
		response.Fail(c, http.StatusUnauthorized, errcode.Unauthorized, errcode.DefaultMessage(errcode.Unauthorized), nil)
		return false
	}
	return true
}

func bunnyWebhook(c *gin.Context) {
	fmt.Println("bunnyWebhook start readBunnyWebhookRawBody")
	rawBody, ok := readBunnyWebhookRawBody(c)
	if !ok {
		return
	}
	if !verifyBunnyWebhookSignature(c, rawBody) {
		return
	}
	req, err := mapping.UnmarshalBunnyVideoWebhookRequestJSON(rawBody)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), nil)
		return
	}
	if err := mapping.ValidateBunnyVideoWebhookRequest(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), nil)
		return
	}

	if err := mediaservice.HandleBunnyVideoWebhook(c.Request.Context(), req); err != nil {
		if pe, ok := errfuncmedia.AsProviderError(err); ok {
			msg := pe.Error()
			response.Fail(c, errfuncmedia.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}

	response.OK(c, "ok", nil)
}
