// Package response provides the unified JSON envelope for all API responses.
//
// Every endpoint returns one of:
//
//	HealthResponse  – GET /health     → { code, message, status }
//	Response        – all other APIs  → { code, message, data }
//
// data may be: null | string | number | boolean | array | object | PaginatedData.
// On error the helpers set data to nil and code to a non-zero errcode constant.
// Typical pattern: pass errcode.DefaultMessage(code) as message — those defaults live in pkg/errcode/messages.go
// and, when the same wording is used for error sentinels, the string is defined once in constants/error_msg.go
// (e.g. MsgFileTooLargeUpload for FileTooLarge + media upload oversize).
//
// # Optional headers and cookies
//
// Every helper accepts an optional [Options] argument as its last parameter.
// Pass it to attach extra response headers or set additional cookies alongside
// the JSON body:
//
//	response.OK(c, "ok", data, response.Options{
//	    Headers: map[string]string{"X-Request-ID": id},
//	    Cookies: map[string]string{"session_hint": "abc"},
//	})
//
// If Options is omitted the behaviour is identical to the previous call without it.
// Cookies set via Options use Path="/", HttpOnly=true, and Secure=false by default.
// For cookies that need full control (maxAge, domain, sameSite …) call [gin.Context.SetCookie]
// directly before or after the response helper.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Options carries optional extra headers and cookies to attach to any response.
// Both fields are plain maps so callers never need to import this package's types
// when they only pass string literals.
//
//	Headers – response header name → value  (e.g. "X-Trace-ID": "abc")
//	Cookies – cookie name → cookie value    (defaults: Path="/", HttpOnly=true, Secure=false)
type Options struct {
	Headers map[string]string
	Cookies map[string]string
}

// Response is the standard envelope for every API endpoint except /health.
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// HealthResponse is the envelope for GET /health.
type HealthResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// PageInfo carries pagination metadata inside PaginatedData.
type PageInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}

// PaginatedData is assigned to Response.Data when the endpoint returns a paginated list.
// result follows the same 6 primitive types as Response.Data (usually an array or null).
type PaginatedData struct {
	Result   any      `json:"result"`
	PageInfo PageInfo `json:"page_info"`
}

// applyOptions sets any extra headers and cookies declared in opts onto c.
// It is a no-op when opts is empty.
func applyOptions(c *gin.Context, opts []Options) {
	if len(opts) == 0 {
		return
	}
	o := opts[0]
	for k, v := range o.Headers {
		c.Header(k, v)
	}
	for name, value := range o.Cookies {
		c.SetCookie(name, value, 0, "/", "", false, true)
	}
}

// Health writes the health-check response (code=0, status="ok").
// Options are accepted for consistency but rarely needed on the health endpoint.
func Health(c *gin.Context, opts ...Options) {
	applyOptions(c, opts)
	c.JSON(http.StatusOK, HealthResponse{Code: 0, Message: "ok", Status: "ok"})
}

// OK writes HTTP 200 with code=0.
func OK(c *gin.Context, message string, data any, opts ...Options) {
	applyOptions(c, opts)
	c.JSON(http.StatusOK, Response{Code: 0, Message: message, Data: data})
}

// Created writes HTTP 201 with code=0.
func Created(c *gin.Context, message string, data any, opts ...Options) {
	applyOptions(c, opts)
	c.JSON(http.StatusCreated, Response{Code: 0, Message: message, Data: data})
}

// OKPaginated writes HTTP 200 with a PaginatedData envelope in the data field.
func OKPaginated(c *gin.Context, message string, result any, pageInfo PageInfo, opts ...Options) {
	applyOptions(c, opts)
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    PaginatedData{Result: result, PageInfo: pageInfo},
	})
}

// Fail writes a JSON error response without aborting the handler chain.
func Fail(c *gin.Context, httpStatus, appCode int, message string, data any, opts ...Options) {
	applyOptions(c, opts)
	c.JSON(httpStatus, Response{Code: appCode, Message: message, Data: data})
}

// AbortFail writes a JSON error response and aborts the handler chain.
func AbortFail(c *gin.Context, httpStatus, appCode int, message string, data any, opts ...Options) {
	applyOptions(c, opts)
	c.AbortWithStatusJSON(httpStatus, Response{Code: appCode, Message: message, Data: data})
}
