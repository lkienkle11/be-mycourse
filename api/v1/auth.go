package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/services/auth"
)

var accessTokenMaxAge = int(constants.AccessTokenTTL.Seconds())

// POST /api/v1/auth/register
func register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	if err := auth.Register(req.Email, req.Password, req.DisplayName); err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrWeakPassword):
			response.Fail(c, http.StatusBadRequest, errcode.WeakPassword, errcode.DefaultMessage(errcode.WeakPassword), nil)
		case errors.Is(err, pkgerrors.ErrEmailAlreadyExists):
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

	result, err := auth.Login(req.Email, req.Password, req.RememberMe)
	if err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrInvalidCredentials):
			response.Fail(c, http.StatusUnauthorized, errcode.InvalidCredentials, errcode.DefaultMessage(errcode.InvalidCredentials), nil)
		case errors.Is(err, pkgerrors.ErrEmailNotConfirmed):
			response.Fail(c, http.StatusUnauthorized, errcode.EmailNotConfirmed, errcode.DefaultMessage(errcode.EmailNotConfirmed), nil)
		case errors.Is(err, pkgerrors.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, errcode.UserDisabled, errcode.DefaultMessage(errcode.UserDisabled), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	setAuthCookies(c, result)
	response.OK(c, "login_success", gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"session_id":    result.SessionStr,
	})
}

// GET /api/v1/auth/confirm?token=<token>
func confirmEmail(c *gin.Context) {
	tok := c.Query("token")
	if tok == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "missing token parameter", nil)
		return
	}

	result, err := auth.ConfirmEmail(tok)
	if err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrInvalidConfirmToken):
			response.Fail(c, http.StatusBadRequest, errcode.InvalidConfirmToken, errcode.DefaultMessage(errcode.InvalidConfirmToken), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	setAuthCookies(c, result)
	response.OK(c, "email_confirmed", gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"session_id":    result.SessionStr,
	})
}

// POST /api/v1/auth/refresh
//
// Rotates the token pair using the refresh token and session ID supplied via
// X-Refresh-Token and X-Session-Id request headers.  On success it returns a
// new access token, a new refresh token, and the (unchanged) session ID in the
// JSON body so the client can update its stored credentials.
func refreshToken(c *gin.Context) {
	refreshTokenStr := c.GetHeader("X-Refresh-Token")
	sessionStr := c.GetHeader("X-Session-Id")

	if refreshTokenStr == "" || sessionStr == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "missing X-Refresh-Token or X-Session-Id header", nil)
		return
	}

	result, err := auth.RefreshSession(sessionStr, refreshTokenStr)
	if err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrRefreshTokenExpired):
			response.Fail(c, http.StatusUnauthorized, errcode.RefreshTokenExpired, errcode.DefaultMessage(errcode.RefreshTokenExpired), nil)
		case errors.Is(err, pkgerrors.ErrInvalidSession), errors.Is(err, pkgerrors.ErrUserNotFound):
			response.Fail(c, http.StatusUnauthorized, errcode.InvalidSession, errcode.DefaultMessage(errcode.InvalidSession), nil)
		case errors.Is(err, pkgerrors.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, errcode.UserDisabled, errcode.DefaultMessage(errcode.UserDisabled), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	response.OK(c, "token_refreshed", gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"session_id":    result.SessionStr,
	})
}

// setAuthCookies writes access_token, refresh_token, and session_id as non-HttpOnly
// SameSite=Lax cookies so the client-side JavaScript layer can read them and attach
// them to requests as Authorization / X-Refresh-Token / X-Session-Id headers.
func setAuthCookies(c *gin.Context, result entities.TokenPairResult) {
	secure := setting.ServerSetting.RunMode == "release"
	refreshMaxAge := int(result.RefreshTTL.Seconds())

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", result.AccessToken, accessTokenMaxAge, "/", "", secure, false)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", result.RefreshToken, refreshMaxAge, "/", "", secure, false)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_id", result.SessionStr, refreshMaxAge, "/", "", secure, false)
}
