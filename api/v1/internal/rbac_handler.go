package internal

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/requestutil"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services/rbac"
)

func listPermissionsInternal(c *gin.Context) {
	var q dto.PermissionFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	rows, total, err := rbac.ListPermissions(dto.ListPermissionsParams{
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
	p, err := rbac.CreatePermission(body.PermissionID, body.PermissionName, body.Description)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", p)
}

func updatePermissionInternal(c *gin.Context) {
	permissionID, ok := requestutil.ParsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid permission id", nil)
		return
	}
	var body dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	p, err := rbac.UpdatePermission(permissionID, body.PermissionID, body.PermissionName, body.Description)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", p)
}

func deletePermissionInternal(c *gin.Context) {
	permissionID, ok := requestutil.ParsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid permission id", nil)
		return
	}
	if err := rbac.DeletePermission(permissionID); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func listRolesInternal(c *gin.Context) {
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	rows, err := rbac.ListRoles(with)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", rows)
}

func getRoleInternal(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	r, err := rbac.GetRole(id, with)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
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
	r, err := rbac.CreateRole(body.Name, body.Description)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", r)
}

func updateRoleInternal(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	var body dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := rbac.UpdateRole(id, body.Name, body.Description)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", r)
}

func setRolePermissionsInternal(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	var body dto.SetRolePermissionsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := rbac.SetRolePermissions(id, body.PermissionIDs)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "role not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", r)
}

func deleteRoleInternal(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	if err := rbac.DeleteRole(id); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}
