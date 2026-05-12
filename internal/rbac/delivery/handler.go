// Package delivery contains the HTTP handlers for the RBAC bounded context.
package delivery

import (
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/rbac/application"
	"mycourse-io-be/internal/rbac/domain" //nolint:depguard // delivery uses domain input types (UpdatePermissionInput, UpdateRoleInput) for mapping; no business logic
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

// Handler holds all HTTP handler methods for the RBAC domain.
type Handler struct {
	svc *application.RBACService
}

// NewHandler constructs an RBAC delivery Handler.
func NewHandler(svc *application.RBACService) *Handler {
	return &Handler{svc: svc}
}

// --- Permissions -------------------------------------------------------------

func (h *Handler) listPermissions(c *gin.Context) {
	var q PermissionFilterRequest
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListPermissions(c.Request.Context(), domain.PermissionFilter{
		Page: q.page(), PageSize: q.perPage(),
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OKPaginated(c, "ok", toPermissionResponses(rows), utils.BuildPage(q.page(), q.perPage(), total))
}

func (h *Handler) createPermission(c *gin.Context) {
	var body CreatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	p, err := h.svc.CreatePermission(c.Request.Context(), domain.CreatePermissionInput{
		PermissionID: body.PermissionID, PermissionName: body.PermissionName, Description: body.Description,
	})
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", toPermissionResponsePtr(p))
}

func (h *Handler) updatePermission(c *gin.Context) { //nolint:dupl // intentional parallel with updateRole; both follow the same HTTP CRUD pattern
	permissionID, ok := utils.ParsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid permission id", nil)
		return
	}
	var body UpdatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	p, err := h.svc.UpdatePermission(c.Request.Context(), permissionID, domain.UpdatePermissionInput{
		PermissionName: body.PermissionName,
		Description:    body.Description,
	})
	if err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toPermissionResponsePtr(p))
}

func (h *Handler) deletePermission(c *gin.Context) {
	permissionID, ok := utils.ParsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid permission id", nil)
		return
	}
	if err := h.svc.DeletePermission(c.Request.Context(), permissionID); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

// --- Roles -------------------------------------------------------------------

func (h *Handler) listRoles(c *gin.Context) {
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	rows, _, err := h.svc.ListRoles(c.Request.Context(), domain.RoleFilter{WithPermissions: with, Page: 1, PageSize: 200})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", toRoleResponses(rows))
}

func (h *Handler) getRole(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	r, err := h.svc.GetRole(c.Request.Context(), id, with)
	if err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", toRoleResponsePtr(r))
}

func (h *Handler) createRole(c *gin.Context) {
	var body CreateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	r, err := h.svc.CreateRole(c.Request.Context(), domain.CreateRoleInput{
		Name: body.Name, Description: body.Description,
	})
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", toRoleResponsePtr(r))
}

func (h *Handler) updateRole(c *gin.Context) { //nolint:dupl // intentional parallel with updatePermission; both follow the same HTTP CRUD pattern
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var body UpdateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	r, err := h.svc.UpdateRole(c.Request.Context(), id, domain.UpdateRoleInput{
		Name: body.Name, Description: body.Description,
	})
	if err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toRoleResponsePtr(r))
}

func (h *Handler) setRolePermissions(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var body SetRolePermissionsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	r, err := h.svc.SetRolePermissions(c.Request.Context(), id, body.PermissionIDs)
	if err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "role not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toRoleResponsePtr(r))
}

func (h *Handler) deleteRole(c *gin.Context) {
	id, ok := utils.ParseUintPathParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteRole(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

// --- User bindings -----------------------------------------------------------

func (h *Handler) listUserRoles(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := h.svc.ListRolesForUser(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", toRoleResponses(rows))
}

func (h *Handler) listUserPermissions(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	set, err := h.svc.PermissionCodesForUser(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	response.OK(c, "ok", UserPermissionCodesResponse{PermissionCodes: list})
}

func (h *Handler) listUserDirectPermissions(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := h.svc.ListPermissionsForUser(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", toPermissionResponses(rows))
}

func (h *Handler) assignUserRole(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	var body AssignUserRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	if err := h.svc.AssignRoleToUser(c.Request.Context(), userID, body.RoleID); err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "role not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "assigned", nil)
}

func (h *Handler) removeUserRole(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	roleID, ok := utils.ParseUintPathParam(c, "roleId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid role id", nil)
		return
	}
	if err := h.svc.RemoveRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}

func (h *Handler) assignUserPermission(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	var body AssignUserPermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	var err error
	switch {
	case body.PermissionID != nil && strings.TrimSpace(*body.PermissionID) != "":
		err = h.svc.AssignPermissionToUser(c.Request.Context(), userID, strings.TrimSpace(*body.PermissionID))
	case body.PermissionName != nil && strings.TrimSpace(*body.PermissionName) != "":
		err = h.svc.AssignPermissionToUserByName(c.Request.Context(), userID, strings.TrimSpace(*body.PermissionName))
	default:
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "permission_id or permission_name required", nil)
		return
	}
	if err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "permission not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "assigned", nil)
}

func (h *Handler) removeUserPermission(c *gin.Context) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	permissionID, ok := utils.ParsePermissionIDParam(c, "permissionId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid permission id", nil)
		return
	}
	if err := h.svc.RemovePermissionFromUser(c.Request.Context(), userID, permissionID); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}

// isNotFound checks if an error is the ErrNotFound sentinel.
func isNotFound(err error) bool {
	return errors.Is(err, apperrors.ErrNotFound)
}
