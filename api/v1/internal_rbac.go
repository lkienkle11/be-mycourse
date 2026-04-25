package v1

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services"
)

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	s := c.Param(name)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}

func parsePermissionIDParam(c *gin.Context, name string) (string, bool) {
	s := strings.TrimSpace(c.Param(name))
	if s == "" || len(s) > 10 {
		return "", false
	}
	return s, true
}

func listPermissionsInternal(c *gin.Context) {
	var q dto.PermissionFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	rows, total, err := services.ListPermissions(services.ListPermissionsParams{
		Offset:     q.GetOffset(),
		Limit:      q.GetPerPage(),
		SortBy:     q.SortBy,
		SortOrder:  q.GetSortOrder(),
		SearchBy:   q.SearchBy,
		SearchData: q.SearchData,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}

	response.OKPaginated(c, "ok", rows, utils.BuildPage(q.GetPage(), q.GetPerPage(), total))
}

func createPermissionInternal(c *gin.Context) {
	var body dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	p, err := services.CreatePermission(body.PermissionID, body.PermissionName, body.Description)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", p)
}

func updatePermissionInternal(c *gin.Context) {
	permissionID, ok := parsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid permission id", nil)
		return
	}
	var body dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	p, err := services.UpdatePermission(permissionID, body.PermissionID, body.PermissionName, body.Description)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", p)
}

func deletePermissionInternal(c *gin.Context) {
	permissionID, ok := parsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid permission id", nil)
		return
	}
	if err := services.DeletePermission(permissionID); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func listRolesInternal(c *gin.Context) {
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	rows, err := services.ListRoles(with)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", rows)
}

func getRoleInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	r, err := services.GetRole(id, with)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", r)
}

func createRoleInternal(c *gin.Context) {
	var body dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := services.CreateRole(body.Name, body.Description)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", r)
}

func updateRoleInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	var body dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := services.UpdateRole(id, body.Name, body.Description)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", r)
}

func setRolePermissionsInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	var body dto.SetRolePermissionsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := services.SetRolePermissions(id, body.PermissionIDs)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "role not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", r)
}

func deleteRoleInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	if err := services.DeleteRole(id); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func listUserRolesInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := services.ListUserRoles(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", rows)
}

func listUserPermissionsInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	set, err := services.PermissionCodesForUser(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	response.OK(c, "ok", list)
}

func assignUserRoleInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	var body dto.AssignUserRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	if err := services.AssignUserRole(userID, body.RoleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "role not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "assigned", nil)
}

func removeUserRoleInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	roleID, ok := parseUintParam(c, "roleId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid role id", nil)
		return
	}
	if err := services.RemoveUserRole(userID, roleID); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}

func listUserDirectPermissionsInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := services.ListUserDirectPermissions(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", rows)
}

func assignUserPermissionInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
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
		err = services.AssignUserPermission(userID, strings.TrimSpace(*body.PermissionID))
	case body.PermissionName != nil && strings.TrimSpace(*body.PermissionName) != "":
		err = services.AssignUserPermissionByPermissionName(userID, strings.TrimSpace(*body.PermissionName))
	default:
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "permission_id or permission_name required", nil)
		return
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "permission not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "assigned", nil)
}

func removeUserPermissionInternal(c *gin.Context) {
	userID, ok := parseUintParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid user id", nil)
		return
	}
	permissionID, ok := parsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid permission id", nil)
		return
	}
	if err := services.RemoveUserPermission(userID, permissionID); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}
