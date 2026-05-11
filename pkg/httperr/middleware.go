package httperr

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/logger"
	appresponse "mycourse-io-be/pkg/response"
	appvalidate "mycourse-io-be/pkg/validate"
)

// Middleware runs after handlers: if nothing was written and c.Errors has entries,
// responds with JSON derived from the last error (Spring @ControllerAdvice-style hook for Gin).
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Written() {
			return
		}
		gerr := c.Errors.Last()
		if gerr == nil {
			return
		}
		respondError(c, gerr.Err)
	}
}

// Recovery replaces gin.Recovery with a JSON body on panic; logs via Zap + stack field.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.FromContext(c.Request.Context()).Error("panic recovered", zap.Any("panic", recovered), zap.Stack("stack"))
		if c.Writer.Written() {
			return
		}
		writeErrorBody(c, http.StatusInternalServerError, errcode.Panic, errcode.DefaultMessage(errcode.Panic), nil)
	})
}

func respondJSONOrValidation(c *gin.Context, err error) bool {
	var syn *json.SyntaxError
	var typ *json.UnmarshalTypeError
	if errors.As(err, &syn) || errors.As(err, &typ) {
		writeErrorBody(c, http.StatusBadRequest, errcode.InvalidJSON, errcode.DefaultMessage(errcode.InvalidJSON), nil)
		return true
	}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := appvalidate.FlattenErrors(ve)
		writeErrorBody(c, http.StatusBadRequest, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), gin.H{"details": details})
		return true
	}
	return false
}

func respondHTTPErrorIfAny(c *gin.Context, err error) bool {
	he, ok := AsHTTPError(err)
	if !ok {
		return false
	}
	appCode := he.AppCode
	if appCode == 0 {
		appCode = errcode.Unknown
	}
	msg := he.Message
	if msg == "" {
		msg = errcode.DefaultMessage(appCode)
	}
	writeErrorBody(c, he.Status, appCode, msg, nil)
	return true
}

func respondError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if respondJSONOrValidation(c, err) {
		return
	}
	if respondHTTPErrorIfAny(c, err) {
		return
	}
	msg := errcode.DefaultMessage(errcode.Unknown)
	if gin.Mode() == gin.DebugMode {
		msg = err.Error()
	}
	writeErrorBody(c, http.StatusInternalServerError, errcode.Unknown, msg, nil)
}

// writeErrorBody writes the unified error envelope: { code, message, data }.
// data is nil for most errors; pass a gin.H for errors that carry extra context (e.g. validation details).
func writeErrorBody(c *gin.Context, httpStatus, appCode int, message string, data any) {
	c.AbortWithStatusJSON(httpStatus, appresponse.Response{
		Code:    appCode,
		Message: message,
		Data:    data,
	})
}
