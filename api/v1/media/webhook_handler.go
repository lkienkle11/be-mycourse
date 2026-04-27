package media

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	pkgmedia "mycourse-io-be/pkg/media"
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
		if pe, ok := pkgmedia.AsProviderError(err); ok {
			msg := pe.Error()
			response.Fail(c, pkgmedia.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}

	response.OK(c, "ok", nil)
}
