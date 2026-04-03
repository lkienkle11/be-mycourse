package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/services"
)

var accessTokenMaxAge = int(services.AccessTokenTTL.Seconds())

// POST /api/v1/auth/register
func register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	if err := services.Register(req.Email, req.Password, req.DisplayName); err != nil {
		switch {
		case errors.Is(err, services.ErrWeakPassword):
			response.Fail(c, http.StatusBadRequest, errcode.WeakPassword, errcode.DefaultMessage(errcode.WeakPassword), nil)
		case errors.Is(err, services.ErrEmailAlreadyExists):
			response.Fail(c, http.StatusConflict, errcode.EmailAlreadyExists, errcode.DefaultMessage(errcode.EmailAlreadyExists), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	response.Created(c, "registration_success", nil)
}

// POST /api/v1/auth/login
func login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	result, err := services.Login(req.Email, req.Password, req.RememberMe)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			response.Fail(c, http.StatusUnauthorized, errcode.InvalidCredentials, errcode.DefaultMessage(errcode.InvalidCredentials), nil)
		case errors.Is(err, services.ErrEmailNotConfirmed):
			response.Fail(c, http.StatusUnauthorized, errcode.EmailNotConfirmed, errcode.DefaultMessage(errcode.EmailNotConfirmed), nil)
		case errors.Is(err, services.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, errcode.UserDisabled, errcode.DefaultMessage(errcode.UserDisabled), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	setAuthCookies(c, result)
	response.OK(c, "login_success", nil)
}

// GET /api/v1/auth/confirm?token=<token>
func confirmEmail(c *gin.Context) {
	tok := c.Query("token")
	if tok == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "missing token parameter", nil)
		return
	}

	result, err := services.ConfirmEmail(tok)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidConfirmToken):
			response.Fail(c, http.StatusBadRequest, errcode.InvalidConfirmToken, errcode.DefaultMessage(errcode.InvalidConfirmToken), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	setAuthCookies(c, result)
	response.OK(c, "email_confirmed", nil)
}

// setAuthCookies writes the access token, refresh token, and session_id as HttpOnly cookies.
// Cookie MaxAge for the refresh token and session_id is derived from result.RefreshTTL so
// remember-me vs non-remember-me sessions get the correct expiry on the client side.
// Tokens are NOT included in the JSON response body.
func setAuthCookies(c *gin.Context, result services.TokenPairResult) {
	secure := setting.ServerSetting.RunMode == "release"
	refreshMaxAge := int(result.RefreshTTL.Seconds())

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("access_token", result.AccessToken, accessTokenMaxAge, "/", "", secure, true)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", result.RefreshToken, refreshMaxAge, "/", "", secure, true)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session_id", result.SessionStr, refreshMaxAge, "/", "", secure, true)
}
