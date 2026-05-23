package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/rbac/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func deleteRBACByID[ID any](
	c *gin.Context,
	parseID func(*gin.Context) (ID, bool),
	deleteFn func(ID) error,
	invalidMsg string,
) {
	id, ok := parseID(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, invalidMsg, nil)
		return
	}
	if err := deleteFn(id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

type userBindingKind int

const (
	userBindingRoles userBindingKind = iota
	userBindingDirectPermissions
)

func (h *Handler) listUserBinding(c *gin.Context, kind userBindingKind) {
	switch kind {
	case userBindingRoles:
		listForUserID(c,
			func(userID uint) ([]domain.Role, error) {
				return h.svc.ListRolesForUser(c.Request.Context(), userID)
			},
			func(rows []domain.Role) any { return toRoleResponses(rows) },
		)
	case userBindingDirectPermissions:
		listForUserID(c,
			func(userID uint) ([]domain.Permission, error) {
				return h.svc.ListPermissionsForUser(c.Request.Context(), userID)
			},
			func(rows []domain.Permission) any { return toPermissionResponses(rows) },
		)
	}
}

func listForUserID[Row any](
	c *gin.Context,
	listFn func(uint) ([]Row, error),
	toResponse func([]Row) any,
) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	rows, err := listFn(userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "ok", toResponse(rows))
}

func removeUserBinding[SecondID any](
	c *gin.Context,
	parseSecond func(*gin.Context) (SecondID, bool),
	removeFn func(userID uint, secondID SecondID) error,
	invalidSecondMsg string,
) {
	userID, ok := utils.ParseUintPathParam(c, "userId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid user id", nil)
		return
	}
	secondID, ok := parseSecond(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, invalidSecondMsg, nil)
		return
	}
	if err := removeFn(userID, secondID); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "removed", nil)
}

func toUpdatePermissionInput(body UpdatePermissionRequest) domain.UpdatePermissionInput {
	return domain.UpdatePermissionInput{PermissionName: body.PermissionName, Description: body.Description}
}

func toUpdateRoleInput(body UpdateRoleRequest) domain.UpdateRoleInput {
	return domain.UpdateRoleInput{Name: body.Name, Description: body.Description}
}

func updateRBACResource(c *gin.Context, h *Handler, kind rbacResourceKind) {
	switch kind {
	case rbacResourcePermission:
		updateByID(c,
			func(c *gin.Context) (string, bool) { return utils.ParsePermissionIDParam(c, "permissionId") },
			"invalid permission id",
			h.permissionUpdater(c),
			toUpdatePermissionInput,
			func(p *domain.Permission) any { return toPermissionResponsePtr(p) },
		)
	case rbacResourceRole:
		updateByID(c,
			func(c *gin.Context) (uint, bool) { return utils.ParseUintPathParam(c, "id") },
			"invalid id",
			h.roleUpdater(c),
			toUpdateRoleInput,
			func(r *domain.Role) any { return toRoleResponsePtr(r) },
		)
	}
}

func rbacCtxUpdater[ID any, In any, Row any](
	c *gin.Context,
	call func(context.Context, ID, In) (*Row, error),
) func(ID, In) (*Row, error) {
	return func(id ID, in In) (*Row, error) {
		return call(c.Request.Context(), id, in)
	}
}

func (h *Handler) permissionUpdater(c *gin.Context) func(string, domain.UpdatePermissionInput) (*domain.Permission, error) {
	return rbacCtxUpdater(c, h.svc.UpdatePermission)
}

func (h *Handler) roleUpdater(c *gin.Context) func(uint, domain.UpdateRoleInput) (*domain.Role, error) {
	return rbacCtxUpdater(c, h.svc.UpdateRole)
}

func deleteRBACResource(c *gin.Context, h *Handler, kind rbacResourceKind) {
	switch kind {
	case rbacResourcePermission:
		deleteRBACByID(c,
			func(c *gin.Context) (string, bool) { return utils.ParsePermissionIDParam(c, "permissionId") },
			h.permissionDeleter(c),
			"invalid permission id",
		)
	case rbacResourceRole:
		deleteRBACByID(c,
			func(c *gin.Context) (uint, bool) { return utils.ParseUintPathParam(c, "id") },
			h.roleDeleter(c),
			"invalid id",
		)
	}
}

func rbacCtxDeleter[ID any](c *gin.Context, call func(context.Context, ID) error) func(ID) error {
	return func(id ID) error { return call(c.Request.Context(), id) }
}

func (h *Handler) permissionDeleter(c *gin.Context) func(string) error {
	return rbacCtxDeleter(c, h.svc.DeletePermission)
}

func (h *Handler) roleDeleter(c *gin.Context) func(uint) error {
	return rbacCtxDeleter(c, h.svc.DeleteRole)
}

func updateByID[ID any, Body any, Row any, In any](
	c *gin.Context,
	parseID func(*gin.Context) (ID, bool),
	invalidIDMsg string,
	updateFn func(ID, In) (*Row, error),
	toInput func(Body) In,
	toResponse func(*Row) any,
) {
	id, ok := parseID(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, invalidIDMsg, nil)
		return
	}
	var body Body
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := updateFn(id, toInput(body))
	if err != nil {
		if isNotFound(err) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toResponse(row))
}
