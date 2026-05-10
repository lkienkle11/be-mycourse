package v1

import (
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/validate"
	"mycourse-io-be/services/auth"
	"mycourse-io-be/services/rbac"
)

// GET /api/v1/me
func getMe(c *gin.Context) {
	v, _ := c.Get(middleware.ContextUserID)
	uid, _ := v.(uint)

	me, err := auth.GetMe(uid)
	if err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "user not found", nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}

	response.OK(c, "ok", mapping.ToMeResponseFromProfile(me))
}

// PATCH /api/v1/me — partial profile update (avatar_file_id).
func patchMe(c *gin.Context) {
	v, _ := c.Get(middleware.ContextUserID)
	uid, _ := v.(uint)

	var req dto.UpdateMeRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	me, err := auth.UpdateMe(uid, req)
	if err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "user not found", nil)
		case errors.Is(err, pkgerrors.ErrInvalidProfileMediaFile):
			response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}
	response.OK(c, "ok", mapping.ToMeResponseFromProfile(me))
}

// GET /api/v1/me/permissions
func getMyPermissions(c *gin.Context) {
	v, _ := c.Get(middleware.ContextUserID)
	uid, _ := v.(uint)
	set, err := rbac.PermissionCodesForUser(uid)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, "failed to load permissions", nil)
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	response.OK(c, "ok", dto.MyPermissionsResponse{Permissions: list})
}
