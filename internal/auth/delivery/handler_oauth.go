package delivery

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/auth/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
)

func (h *Handler) writeOAuthTokenSuccess(c *gin.Context, result domain.TokenPairResult, message string) {
	h.setAuthCookies(c, result.AccessToken, result.RefreshToken, result.SessionStr, result.RefreshTTL.Seconds())
	response.OK(c, message, toTokensResponse(result))
}

func (h *Handler) writeOAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidGoogleCode):
		response.Fail(c, http.StatusUnauthorized, apperrors.InvalidGoogleCode, apperrors.DefaultMessage(apperrors.InvalidGoogleCode), nil)
	case errors.Is(err, domain.ErrGoogleEmailNotVerified):
		response.Fail(c, http.StatusBadRequest, apperrors.GoogleEmailNotVerified, apperrors.DefaultMessage(apperrors.GoogleEmailNotVerified), nil)
	case errors.Is(err, domain.ErrOAuthIdentityConflict):
		response.Fail(c, http.StatusConflict, apperrors.OAuthIdentityConflict, apperrors.DefaultMessage(apperrors.OAuthIdentityConflict), nil)
	case errors.Is(err, domain.ErrInvalidXCode):
		response.Fail(c, http.StatusUnauthorized, apperrors.InvalidXCode, apperrors.DefaultMessage(apperrors.InvalidXCode), nil)
	case errors.Is(err, domain.ErrXEmailUnavailable):
		response.Fail(c, http.StatusBadRequest, apperrors.XEmailUnavailable, apperrors.DefaultMessage(apperrors.XEmailUnavailable), nil)
	case errors.Is(err, domain.ErrXAccountLinkRequired):
		response.Fail(c, http.StatusConflict, apperrors.XAccountLinkRequired, apperrors.DefaultMessage(apperrors.XAccountLinkRequired), nil)
	case errors.Is(err, domain.ErrInvalidDiscordCode):
		response.Fail(c, http.StatusUnauthorized, apperrors.InvalidDiscordCode, apperrors.DefaultMessage(apperrors.InvalidDiscordCode), nil)
	case errors.Is(err, domain.ErrDiscordEmailNotVerified):
		response.Fail(c, http.StatusBadRequest, apperrors.DiscordEmailNotVerified, apperrors.DefaultMessage(apperrors.DiscordEmailNotVerified), nil)
	case errors.Is(err, domain.ErrDiscordEmailUnavailable):
		response.Fail(c, http.StatusBadRequest, apperrors.DiscordEmailUnavailable, apperrors.DefaultMessage(apperrors.DiscordEmailUnavailable), nil)
	case errors.Is(err, domain.ErrInvalidCredentials):
		response.Fail(c, http.StatusUnauthorized, apperrors.InvalidCredentials, apperrors.DefaultMessage(apperrors.InvalidCredentials), nil)
	case errors.Is(err, domain.ErrUserDisabled):
		response.Fail(c, http.StatusForbidden, apperrors.UserDisabled, apperrors.DefaultMessage(apperrors.UserDisabled), nil)
	case errors.Is(err, domain.ErrUserBanned):
		response.Fail(c, http.StatusForbidden, apperrors.UserBanned, apperrors.DefaultMessage(apperrors.UserBanned), nil)
	default:
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	}
}

// GoogleLogin — POST /api/v1/auth/google
func (h *Handler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	result, err := h.auth.GoogleLoginFromCode(c.Request.Context(), req.Code, req.RememberMe)
	if err != nil {
		h.writeOAuthError(c, err)
		return
	}
	h.writeOAuthTokenSuccess(c, result, "google_login_success")
}

type googleTokenRequest interface {
	TokenValue() string
}

func bindGoogleTokenRequest[T googleTokenRequest](c *gin.Context) (string, error) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		return "", err
	}
	return req.TokenValue(), nil
}

func (h *Handler) handleGoogleIDTokenLogin(
	c *gin.Context,
	bind func(*gin.Context) (string, error),
	login func(context.Context, string) (domain.TokenPairResult, error),
	successMessage string,
) {
	token, err := bind(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	result, err := login(c.Request.Context(), token)
	if err != nil {
		h.writeOAuthError(c, err)
		return
	}
	h.writeOAuthTokenSuccess(c, result, successMessage)
}

// GoogleOneTap — POST /api/v1/auth/google/onetap
func (h *Handler) GoogleOneTap(c *gin.Context) {
	h.handleGoogleIDTokenLogin(
		c,
		bindGoogleTokenRequest[GoogleOneTapRequest],
		h.auth.GoogleLoginFromCredential,
		"google_onetap_success",
	)
}

// GoogleMobile — POST /api/v1/auth/google/mobile
func (h *Handler) GoogleMobile(c *gin.Context) {
	h.handleGoogleIDTokenLogin(
		c,
		bindGoogleTokenRequest[GoogleMobileRequest],
		h.auth.GoogleLoginFromIDToken,
		"google_mobile_success",
	)
}

// XLogin — POST /api/v1/auth/x
func (h *Handler) XLogin(c *gin.Context) {
	var req XLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	entrypoint := req.EntryPoint
	if entrypoint == "" {
		entrypoint = "login"
	}
	result, err := h.auth.XLoginFromCode(c.Request.Context(), req.Code, req.CodeVerifier, entrypoint, req.RememberMe)
	if err != nil {
		h.writeOAuthError(c, err)
		return
	}
	h.writeOAuthTokenSuccess(c, result, "x_login_success")
}

// DiscordLogin — POST /api/v1/auth/discord
func (h *Handler) DiscordLogin(c *gin.Context) {
	var req DiscordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	entrypoint := req.EntryPoint
	if entrypoint == "" {
		entrypoint = "login"
	}
	result, err := h.auth.DiscordLoginFromCode(c.Request.Context(), req.Code, entrypoint, req.RememberMe)
	if err != nil {
		h.writeOAuthError(c, err)
		return
	}
	h.writeOAuthTokenSuccess(c, result, "discord_login_success")
}
