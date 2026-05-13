package delivery

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/auth/domain" //nolint:depguard // delivery uses domain error sentinels for errors.Is checks; no business logic here
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/validate"
)

// AuthUseCase is implemented by application.AuthService.
type AuthUseCase interface {
	Login(ctx context.Context, email, password string, rememberMe bool) (domain.TokenPairResult, error)
	Register(ctx context.Context, email, password, displayName string) error
	ConfirmEmail(ctx context.Context, confirmToken string) (domain.TokenPairResult, error)
	RefreshSession(ctx context.Context, sessionStr, refreshTokenStr string) (domain.TokenPairResult, error)
	GetMe(ctx context.Context, userID uint) (*domain.MeProfile, error)
	UpdateMe(ctx context.Context, userID uint, avatarFileID *string) (*domain.MeProfile, error)
	SoftDeleteUser(ctx context.Context, userID uint) error
}

// PermissionUseCase provides a user's permissions set (used by /me/permissions).
type PermissionUseCase interface {
	PermissionCodesForUser(userID uint) (map[string]struct{}, error)
}

var accessTokenMaxAge = int(domain.AccessTokenTTL.Seconds())

// Handler holds the auth HTTP handlers.
type Handler struct {
	auth AuthUseCase
	perm PermissionUseCase
}

// NewHandler constructs the auth HTTP handler.
func NewHandler(auth AuthUseCase, perm PermissionUseCase) *Handler {
	return &Handler{auth: auth, perm: perm}
}

// CSRFToken — GET /api/v1/auth/csrf
func (h *Handler) CSRFToken(c *gin.Context) {
	tok, err := c.Cookie(middleware.CookieCSRFToken)
	if err != nil || tok == "" {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, "failed to load csrf token", nil)
		return
	}
	response.OK(c, "csrf_token_issued", CSRFTokenResponse{CSRFToken: tok})
}

// Register — POST /api/v1/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	if err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName); err != nil {
		h.writeRegisterError(c, err)
		return
	}
	response.Created(c, "registration_success", nil)
}

func (h *Handler) writeRegisterError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrWeakPassword):
		response.Fail(c, http.StatusBadRequest, apperrors.WeakPassword, apperrors.DefaultMessage(apperrors.WeakPassword), nil)
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		response.Fail(c, http.StatusConflict, apperrors.EmailAlreadyExists, apperrors.DefaultMessage(apperrors.EmailAlreadyExists), nil)
	case errors.Is(err, domain.ErrRegistrationAbandoned):
		response.Fail(c, http.StatusGone, apperrors.RegistrationAbandoned, apperrors.DefaultMessage(apperrors.RegistrationAbandoned), nil)
	case errors.Is(err, domain.ErrConfirmationEmailSendFailed):
		response.Fail(c, http.StatusBadGateway, apperrors.ConfirmationEmailSendFailed, apperrors.DefaultMessage(apperrors.ConfirmationEmailSendFailed), nil)
	default:
		var rl *domain.RegistrationEmailRateLimitedError
		if errors.As(err, &rl) {
			sec := strconv.FormatInt(rl.RetryAfterSeconds, 10)
			response.Fail(c, http.StatusTooManyRequests, apperrors.RegistrationEmailRateLimited, apperrors.DefaultMessage(apperrors.RegistrationEmailRateLimited), nil,
				response.Options{Headers: map[string]string{
					constants.HeaderRegisterRetryAfter:         sec,
					constants.HeaderRegisterRetryAfterExtended: sec,
				}})
			return
		}
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	}
}

// Login — POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, req.RememberMe)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			response.Fail(c, http.StatusUnauthorized, apperrors.InvalidCredentials, apperrors.DefaultMessage(apperrors.InvalidCredentials), nil)
		case errors.Is(err, domain.ErrEmailNotConfirmed):
			response.Fail(c, http.StatusUnauthorized, apperrors.EmailNotConfirmed, apperrors.DefaultMessage(apperrors.EmailNotConfirmed), nil)
		case errors.Is(err, domain.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, apperrors.UserDisabled, apperrors.DefaultMessage(apperrors.UserDisabled), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		}
		return
	}
	h.setAuthCookies(c, result.AccessToken, result.RefreshToken, result.SessionStr, result.RefreshTTL.Seconds())
	response.OK(c, "login_success", toTokensResponse(result))
}

// ConfirmEmail — POST /api/v1/auth/confirm
func (h *Handler) ConfirmEmail(c *gin.Context) {
	var req ConfirmEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	result, err := h.auth.ConfirmEmail(c.Request.Context(), req.Token)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidConfirmToken):
			response.Fail(c, http.StatusBadRequest, apperrors.InvalidConfirmToken, apperrors.DefaultMessage(apperrors.InvalidConfirmToken), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		}
		return
	}
	h.setAuthCookies(c, result.AccessToken, result.RefreshToken, result.SessionStr, result.RefreshTTL.Seconds())
	response.OK(c, "email_confirmed", toTokensResponse(result))
}

// RefreshToken — POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshTokenStr := c.GetHeader("X-Refresh-Token")
	sessionStr := c.GetHeader("X-Session-Id")
	if refreshTokenStr == "" || sessionStr == "" {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "missing X-Refresh-Token or X-Session-Id header", nil)
		return
	}
	result, err := h.auth.RefreshSession(c.Request.Context(), sessionStr, refreshTokenStr)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRefreshTokenExpired):
			response.Fail(c, http.StatusUnauthorized, apperrors.RefreshTokenExpired, apperrors.DefaultMessage(apperrors.RefreshTokenExpired), nil)
		case errors.Is(err, domain.ErrInvalidSession), errors.Is(err, domain.ErrUserNotFound):
			response.Fail(c, http.StatusUnauthorized, apperrors.InvalidSession, apperrors.DefaultMessage(apperrors.InvalidSession), nil)
		case errors.Is(err, domain.ErrUserDisabled):
			response.Fail(c, http.StatusForbidden, apperrors.UserDisabled, apperrors.DefaultMessage(apperrors.UserDisabled), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		}
		return
	}
	response.OK(c, "token_refreshed", toTokensResponse(result))
}

// GetMe — GET /api/v1/me
func (h *Handler) GetMe(c *gin.Context) {
	uid := currentUserID(c)
	me, err := h.auth.GetMe(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "user not found", nil)
		} else {
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		}
		return
	}
	response.OK(c, "ok", toMeResponse(me))
}

// PatchMe — PATCH /api/v1/me
func (h *Handler) PatchMe(c *gin.Context) {
	uid := currentUserID(c)
	var req UpdateMeRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	me, err := h.auth.UpdateMe(c.Request.Context(), uid, req.AvatarFileID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "user not found", nil)
		default:
			response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		}
		return
	}
	response.OK(c, "ok", toMeResponse(me))
}

// DeleteMe — DELETE /api/v1/me
func (h *Handler) DeleteMe(c *gin.Context) {
	uid := currentUserID(c)
	if err := h.auth.SoftDeleteUser(c.Request.Context(), uid); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "user not found", nil)
		} else {
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		}
		return
	}
	response.OK(c, "account_deleted", nil)
}

// GetMyPermissions — GET /api/v1/me/permissions
func (h *Handler) GetMyPermissions(c *gin.Context) {
	uid := currentUserID(c)
	set, err := h.perm.PermissionCodesForUser(uid)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, "failed to load permissions", nil)
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	response.OK(c, "ok", MyPermissionsResponse{Permissions: list})
}

// --- helpers ---

func currentUserID(c *gin.Context) uint {
	v, _ := c.Get(middleware.ContextUserID)
	uid, _ := v.(uint)
	return uid
}

func toTokensResponse(r domain.TokenPairResult) LoginSessionTokensResponse {
	return LoginSessionTokensResponse{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		SessionID:    r.SessionStr,
	}
}

func toMeResponse(me *domain.MeProfile) *MeResponse {
	if me == nil {
		return nil
	}
	return &MeResponse{
		UserID:          me.UserID,
		UserCode:        me.UserCode,
		Email:           me.Email,
		DisplayName:     me.DisplayName,
		AvatarURL:       me.AvatarURL,
		AvatarObjectKey: me.AvatarObjectKey,
		EmailConfirmed:  me.EmailConfirmed,
		IsDisabled:      me.IsDisabled,
		CreatedAt:       me.CreatedAt,
		Permissions:     me.Permissions,
	}
}

func (h *Handler) setAuthCookies(c *gin.Context, accessToken, refreshToken, sessionID string, refreshTTLSeconds float64) {
	secure := setting.ServerSetting.RunMode == "release"
	refreshMaxAge := int(refreshTTLSeconds)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", accessToken, accessTokenMaxAge, "/", "", secure, false)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", refreshToken, refreshMaxAge, "/", "", secure, false)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_id", sessionID, refreshMaxAge, "/", "", secure, false)
}
