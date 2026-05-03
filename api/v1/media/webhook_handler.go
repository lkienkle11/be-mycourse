package media

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/response"
	mediaservice "mycourse-io-be/services/media"
)

func bunnyWebhook(c *gin.Context) {
	var req dto.BunnyVideoWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), nil)
		return
	}

	if err := mediaservice.HandleBunnyVideoWebhook(c.Request.Context(), req); err != nil {
		if pe, ok := pkgerrors.AsProviderError(err); ok {
			msg := pe.Error()
			response.Fail(c, pkgerrors.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}

	response.OK(c, "ok", nil)
}
