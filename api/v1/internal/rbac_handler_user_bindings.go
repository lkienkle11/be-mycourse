package internal

import (
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/requestutil"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services/rbac"
)

func listUserRolesInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := rbac.ListUserRoles(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", mapping.ToRBACRoleResponses(rows))
}

func listUserPermissionsInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	set, err := rbac.PermissionCodesForUser(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	response.OK(c, "ok", mapping.ToUserRBACPermissionCodesResponse(list))
}

func assignUserRoleInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	var body dto.AssignUserRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	if err := rbac.AssignUserRole(userID, body.RoleID); err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "role not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "assigned", nil)
}

func removeUserRoleInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	roleID, ok := utils.ParseUintPathParam(c, "roleId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid role id", nil)
		return
	}
	if err := rbac.RemoveUserRole(userID, roleID); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}

func listUserDirectPermissionsInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := rbac.ListUserDirectPermissions(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", mapping.ToRBACPermissionResponses(rows))
}

func assignUserPermissionInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	var body dto.AssignUserPermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	var err error
	switch {
	case body.PermissionID != nil && strings.TrimSpace(*body.PermissionID) != "":
		err = rbac.AssignUserPermission(userID, strings.TrimSpace(*body.PermissionID))
	case body.PermissionName != nil && strings.TrimSpace(*body.PermissionName) != "":
		err = rbac.AssignUserPermissionByPermissionName(userID, strings.TrimSpace(*body.PermissionName))
	default:
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "permission_id or permission_name required", nil)
		return
	}
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "permission not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "assigned", nil)
}

func removeUserPermissionInternal(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	permissionID, ok := requestutil.ParsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid permission id", nil)
		return
	}
	if err := rbac.RemoveUserPermission(userID, permissionID); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}
