package httperr

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"mycourse-io-be/pkg/errcode"
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

// Recovery replaces gin.Recovery with a JSON body on panic (still logs stack in debug).
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("panic: %v", recovered)
		if c.Writer.Written() {
			return
		}
		writeErrorBody(c, http.StatusInternalServerError, errcode.Panic, errcode.DefaultMessage(errcode.Panic), nil, nil)
	})
}

func respondError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var syn *json.SyntaxError
	var typ *json.UnmarshalTypeError
	if errors.As(err, &syn) || errors.As(err, &typ) {
		writeErrorBody(c, http.StatusBadRequest, errcode.InvalidJSON, errcode.DefaultMessage(errcode.InvalidJSON), nil, nil)
		return
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := appvalidate.FlattenErrors(ve)
		writeErrorBody(c, http.StatusBadRequest, errcode.ValidationFailed, errcode.DefaultMessage(errcode.ValidationFailed), nil, gin.H{"details": details})
		return
	}

	if he, ok := AsHTTPError(err); ok {
		appCode := he.AppCode
		if appCode == 0 {
			appCode = errcode.Unknown
		}
		msg := he.Message
		if msg == "" {
			msg = errcode.DefaultMessage(appCode)
		}
		var extra gin.H
		if he.Err != nil && gin.Mode() == gin.DebugMode {
			extra = gin.H{"cause": he.Err.Error()}
		}
		writeErrorBody(c, he.Status, appCode, msg, errorKeyPtr(he.Code), extra)
		return
	}

	msg := errcode.DefaultMessage(errcode.Unknown)
	if gin.Mode() == gin.DebugMode {
		msg = err.Error()
	}
	writeErrorBody(c, http.StatusInternalServerError, errcode.Unknown, msg, nil, nil)
}

func errorKeyPtr(key string) *string {
	if key == "" {
		return nil
	}
	return &key
}

// writeErrorBody sets JSON: error_code, message, optional error_key, merges extra.
func writeErrorBody(c *gin.Context, httpStatus, appCode int, message string, errorKey *string, extra gin.H) {
	body := gin.H{
		"error_code": appCode,
		"message":    message,
	}
	if errorKey != nil {
		body["error_key"] = *errorKey
	}
	for k, v := range extra {
		body[k] = v
	}
	c.AbortWithStatusJSON(httpStatus, body)
}
