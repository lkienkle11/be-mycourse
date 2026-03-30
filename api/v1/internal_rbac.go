package v1

import (
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/httperr"
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

func listPermissionsInternal(c *gin.Context) {
	rows, err := services.ListPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func createPermissionInternal(c *gin.Context) {
	var body dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	p, err := services.CreatePermission(body.Code, body.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func updatePermissionInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var body dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	p, err := services.UpdatePermission(id, body.Code, body.Description)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func deletePermissionInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := services.DeletePermission(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func listRolesInternal(c *gin.Context) {
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	withParent := c.Query("with_parent") == "1" || c.Query("with_parent") == "true"
	withChildren := c.Query("with_children") == "1" || c.Query("with_children") == "true"
	rows, err := services.ListRoles(with, withParent, withChildren)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func getRoleInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	with := c.Query("with_permissions") == "1" || c.Query("with_permissions") == "true"
	withParent := c.Query("with_parent") == "1" || c.Query("with_parent") == "true"
	withChildren := c.Query("with_children") == "1" || c.Query("with_children") == "true"
	r, err := services.GetRole(id, with, withParent, withChildren)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, r)
}

func createRoleInternal(c *gin.Context) {
	var body dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := services.CreateRole(body.Name, body.Description, body.ParentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, r)
}

func updateRoleInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var body dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := services.UpdateRole(id, body.Name, body.Description, body.ParentID, body.RemoveParent)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, r)
}

func setRolePermissionsInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var body dto.SetRolePermissionsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	r, err := services.SetRolePermissions(id, body.PermissionCodes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "role not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, r)
}

func deleteRoleInternal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := services.DeleteRole(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func listUserRolesInternal(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing user id"})
		return
	}
	rows, err := services.ListUserRoles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func listUserPermissionsInternal(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing user id"})
		return
	}
	set, err := services.PermissionCodesForUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	list := make([]string, 0, len(set))
	for code := range set {
		list = append(list, code)
	}
	sort.Strings(list)
	c.JSON(http.StatusOK, gin.H{"permissions": list})
}

func assignUserRoleInternal(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing user id"})
		return
	}
	var body dto.AssignUserRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httperr.Abort(c, err)
		return
	}
	if err := services.AssignUserRole(userID, body.RoleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "role not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func removeUserRoleInternal(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing user id"})
		return
	}
	roleID, ok := parseUintParam(c, "roleId")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid role id"})
		return
	}
	if err := services.RemoveUserRole(userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
