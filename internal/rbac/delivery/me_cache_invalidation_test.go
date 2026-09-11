package delivery

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/rbac/application"
	"mycourse-io-be/internal/rbac/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

const roleBindingTestUserID = "00000000-0000-4000-8000-000000000001"

type roleBindingRepoStub struct {
	assignErr error
	removeErr error
}

func (s *roleBindingRepoStub) ListRolesForUser(context.Context, string) ([]domain.Role, error) {
	return nil, nil
}

func (s *roleBindingRepoStub) AssignRole(context.Context, string, uint) error { return s.assignErr }

func (s *roleBindingRepoStub) RemoveRole(context.Context, string, uint) error { return s.removeErr }

type permissionBindingRepoStub struct {
	assignErr error
	removeErr error
}

func (*permissionBindingRepoStub) ListPermissionsForUser(context.Context, string) ([]domain.Permission, error) {
	return nil, nil
}

func (*permissionBindingRepoStub) PermissionCodesForUser(context.Context, string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func (*permissionBindingRepoStub) PermissionCodesForUsers(context.Context, []string) (map[string]map[string]struct{}, error) {
	return map[string]map[string]struct{}{}, nil
}

func (s *permissionBindingRepoStub) AssignPermission(context.Context, string, string) error {
	return s.assignErr
}

func (s *permissionBindingRepoStub) AssignPermissionByName(context.Context, string, string) error {
	return s.assignErr
}

func (s *permissionBindingRepoStub) RemovePermission(context.Context, string, string) error {
	return s.removeErr
}

type meCacheInvalidatorSpy struct {
	userIDs []string
}

func (s *meCacheInvalidatorSpy) InvalidateUserMeCache(_ context.Context, userID string) {
	s.userIDs = append(s.userIDs, userID)
}

func newRoleBindingTestRouter(repo *roleBindingRepoStub, cache *meCacheInvalidatorSpy) *gin.Engine {
	gin.SetMode(gin.TestMode)
	service := application.NewRBACService(nil, nil, repo, nil)
	handler := NewHandler(service, cache)
	router := gin.New()
	RegisterRoutes(router.Group("/api/internal-v1"), handler)
	return router
}

func newPermissionBindingTestRouter(repo *permissionBindingRepoStub, cache *meCacheInvalidatorSpy) *gin.Engine {
	gin.SetMode(gin.TestMode)
	service := application.NewRBACService(nil, nil, &roleBindingRepoStub{}, repo)
	handler := NewHandler(service, cache)
	router := gin.New()
	RegisterRoutes(router.Group("/api/internal-v1"), handler)
	return router
}

func assertBindingMutation(
	t *testing.T,
	router *gin.Engine,
	cache *meCacheInvalidatorSpy,
	method string,
	path string,
	body string,
	wantStatus int,
	wantCalls int,
) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, wantStatus, res.Body.String())
	}
	if len(cache.userIDs) != wantCalls {
		t.Fatalf("invalidations = %v, want %d calls", cache.userIDs, wantCalls)
	}
	if wantCalls == 1 && cache.userIDs[0] != roleBindingTestUserID {
		t.Fatalf("invalidated user = %q", cache.userIDs[0])
	}
}

type bindingMutationCase struct {
	name        string
	path        string
	body        string
	mutationErr error
	wantStatus  int
	wantCalls   int
}

func runBindingMutationCases(
	t *testing.T,
	tests []bindingMutationCase,
	method string,
	newRouter func(error, *meCacheInvalidatorSpy) *gin.Engine,
) {
	t.Helper()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cache := &meCacheInvalidatorSpy{}
			router := newRouter(tc.mutationErr, cache)
			assertBindingMutation(t, router, cache, method, tc.path, tc.body, tc.wantStatus, tc.wantCalls)
		})
	}
}

func TestAssignUserRoleInvalidatesMeCacheOnlyAfterSuccess(t *testing.T) {
	t.Parallel()

	tests := []bindingMutationCase{
		{name: "success", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles", body: `{"role_id":2}`, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "invalid user", path: "/api/internal-v1/rbac/users/not-a-uuid/roles", body: `{"role_id":2}`, wantStatus: http.StatusBadRequest},
		{name: "invalid body", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "role not found", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles", body: `{"role_id":2}`, mutationErr: apperrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "persistence failure", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles", body: `{"role_id":2}`, mutationErr: errors.New("assign failed"), wantStatus: http.StatusBadRequest},
	}
	runBindingMutationCases(t, tests, http.MethodPost, func(err error, cache *meCacheInvalidatorSpy) *gin.Engine {
		return newRoleBindingTestRouter(&roleBindingRepoStub{assignErr: err}, cache)
	})
}

func TestRemoveUserRoleInvalidatesMeCacheOnlyAfterSuccess(t *testing.T) {
	t.Parallel()

	tests := []bindingMutationCase{
		{name: "success", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles/2", wantStatus: http.StatusOK, wantCalls: 1},
		{name: "invalid role", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles/not-a-role", wantStatus: http.StatusBadRequest},
		{name: "binding not found", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles/2", mutationErr: apperrors.ErrNotFound, wantStatus: http.StatusInternalServerError},
		{name: "persistence failure", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/roles/2", mutationErr: errors.New("remove failed"), wantStatus: http.StatusInternalServerError},
	}
	runBindingMutationCases(t, tests, http.MethodDelete, func(err error, cache *meCacheInvalidatorSpy) *gin.Engine {
		return newRoleBindingTestRouter(&roleBindingRepoStub{removeErr: err}, cache)
	})
}

func TestAssignUserPermissionInvalidatesMeCacheOnlyAfterSuccess(t *testing.T) {
	t.Parallel()

	tests := []bindingMutationCase{
		{name: "permission id success", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions", body: `{"permission_id":"P1"}`, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "permission name success", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions", body: `{"permission_name":"user:read"}`, wantStatus: http.StatusOK, wantCalls: 1},
		{name: "invalid user", path: "/api/internal-v1/rbac/users/not-a-uuid/direct-permissions", body: `{"permission_id":"P1"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid body", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing permission", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "permission not found", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions", body: `{"permission_id":"P1"}`, mutationErr: apperrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "persistence failure", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions", body: `{"permission_id":"P1"}`, mutationErr: errors.New("assign failed"), wantStatus: http.StatusBadRequest},
	}
	runBindingMutationCases(t, tests, http.MethodPost, func(err error, cache *meCacheInvalidatorSpy) *gin.Engine {
		return newPermissionBindingTestRouter(&permissionBindingRepoStub{assignErr: err}, cache)
	})
}

func TestRemoveUserPermissionInvalidatesMeCacheOnlyAfterSuccess(t *testing.T) {
	t.Parallel()

	tests := []bindingMutationCase{
		{name: "success", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions/P1", wantStatus: http.StatusOK, wantCalls: 1},
		{name: "invalid user", path: "/api/internal-v1/rbac/users/not-a-uuid/direct-permissions/P1", wantStatus: http.StatusBadRequest},
		{name: "invalid permission", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions/permission-id-too-long", wantStatus: http.StatusBadRequest},
		{name: "binding not found", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions/P1", mutationErr: apperrors.ErrNotFound, wantStatus: http.StatusInternalServerError},
		{name: "persistence failure", path: "/api/internal-v1/rbac/users/" + roleBindingTestUserID + "/direct-permissions/P1", mutationErr: errors.New("remove failed"), wantStatus: http.StatusInternalServerError},
	}
	runBindingMutationCases(t, tests, http.MethodDelete, func(err error, cache *meCacheInvalidatorSpy) *gin.Engine {
		return newPermissionBindingTestRouter(&permissionBindingRepoStub{removeErr: err}, cache)
	})
}
