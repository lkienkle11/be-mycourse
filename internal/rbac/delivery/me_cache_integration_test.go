package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	authapp "mycourse-io-be/internal/auth/application"
	authdelivery "mycourse-io-be/internal/auth/delivery"
	authdomain "mycourse-io-be/internal/auth/domain"
	rbacapp "mycourse-io-be/internal/rbac/application"
	rbacdomain "mycourse-io-be/internal/rbac/domain"
	"mycourse-io-be/internal/shared/middleware"
)

type meCacheIntegrationUserRepo struct {
	user *authdomain.User
}

func (r *meCacheIntegrationUserRepo) FindByID(context.Context, string) (*authdomain.User, error) {
	return r.user, nil
}

func (*meCacheIntegrationUserRepo) FindByEmail(context.Context, string) (*authdomain.User, error) {
	return nil, nil
}

func (*meCacheIntegrationUserRepo) FindByUserCode(context.Context, string) (*authdomain.User, error) {
	return nil, nil
}

func (*meCacheIntegrationUserRepo) FindByConfirmationToken(context.Context, string) (*authdomain.User, error) {
	return nil, nil
}

func (*meCacheIntegrationUserRepo) Create(context.Context, *authdomain.User) error { return nil }
func (*meCacheIntegrationUserRepo) Save(context.Context, *authdomain.User) error   { return nil }

func (*meCacheIntegrationUserRepo) UpdateDisplayName(context.Context, string, string) error {
	return nil
}

func (*meCacheIntegrationUserRepo) UpdateAvatar(context.Context, string, *string) error {
	return nil
}

func (*meCacheIntegrationUserRepo) SoftDelete(context.Context, string) error { return nil }
func (*meCacheIntegrationUserRepo) HardDelete(context.Context, string) error { return nil }

type meCacheIntegrationRoleRepo struct {
	mu      sync.Mutex
	roleIDs []uint
	roles   map[uint]string
	listErr error
}

func (r *meCacheIntegrationRoleRepo) ListRolesForUser(context.Context, string) ([]rbacdomain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	roles := make([]rbacdomain.Role, len(r.roleIDs))
	for i, roleID := range r.roleIDs {
		roles[i] = rbacdomain.Role{ID: roleID, Name: r.roles[roleID]}
	}
	return roles, nil
}

func (r *meCacheIntegrationRoleRepo) AssignRole(_ context.Context, _ string, roleID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, assignedID := range r.roleIDs {
		if assignedID == roleID {
			return nil
		}
	}
	r.roleIDs = append(r.roleIDs, roleID)
	return nil
}

func (r *meCacheIntegrationRoleRepo) RemoveRole(_ context.Context, _ string, roleID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, assignedID := range r.roleIDs {
		if assignedID == roleID {
			r.roleIDs = append(r.roleIDs[:i], r.roleIDs[i+1:]...)
			break
		}
	}
	return nil
}

func (r *meCacheIntegrationRoleRepo) RoleNamesForUser(context.Context, string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	names := make([]string, len(r.roleIDs))
	for i, roleID := range r.roleIDs {
		names[i] = r.roles[roleID]
	}
	return names, nil
}

type meCacheIntegrationRedisHook struct {
	mu     sync.Mutex
	values map[string]string
}

func (h *meCacheIntegrationRedisHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *meCacheIntegrationRedisHook) ProcessHook(_ redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		args := cmd.Args()
		switch cmd.Name() {
		case "get":
			key := fmt.Sprint(args[1])
			value, ok := h.values[key]
			if !ok {
				cmd.SetErr(redis.Nil)
				return redis.Nil
			}
			cmd.(*redis.StringCmd).SetVal(value)
			return nil
		case "set":
			h.values[fmt.Sprint(args[1])] = fmt.Sprint(args[2])
			cmd.(*redis.StatusCmd).SetVal("OK")
			return nil
		case "del":
			var removed int64
			for _, arg := range args[1:] {
				key := fmt.Sprint(arg)
				if _, ok := h.values[key]; ok {
					delete(h.values, key)
					removed++
				}
			}
			cmd.(*redis.IntCmd).SetVal(removed)
			return nil
		default:
			return fmt.Errorf("unexpected Redis command %q", cmd.Name())
		}
	}
}

func (h *meCacheIntegrationRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

type meCacheIntegrationFixture struct {
	router   *gin.Engine
	roleRepo *meCacheIntegrationRoleRepo
	cache    *meCacheIntegrationRedisHook
}

func newMeCacheIntegrationFixture(t *testing.T) *meCacheIntegrationFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	roleRepo := &meCacheIntegrationRoleRepo{
		roleIDs: []uint{4},
		roles:   map[uint]string{2: "admin", 4: "learner"},
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	cache := &meCacheIntegrationRedisHook{values: make(map[string]string)}
	redisClient.AddHook(cache)
	t.Cleanup(func() { _ = redisClient.Close() })

	authService := authapp.NewAuthService(
		&meCacheIntegrationUserRepo{user: &authdomain.User{ID: roleBindingTestUserID, Email: "user@example.com"}},
		nil,
		authapp.MeProjectionDeps{Roles: roleRepo},
		nil,
		nil,
		nil,
		redisClient,
	)
	rbacService := rbacapp.NewRBACService(nil, nil, roleRepo, nil)
	rbacHandler := NewHandler(rbacService, authService)
	authHandler := authdelivery.NewHandler(authService, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, roleBindingTestUserID)
		c.Next()
	})
	router.GET("/api/v1/me", authHandler.GetMe)
	router.POST("/api/internal-v1/rbac/users/:userId/roles", rbacHandler.assignUserRole)
	router.DELETE("/api/internal-v1/rbac/users/:userId/roles/:roleId", rbacHandler.removeUserRole)
	return &meCacheIntegrationFixture{router: router, roleRepo: roleRepo, cache: cache}
}

func (f *meCacheIntegrationFixture) assertRoles(t *testing.T, want []string) {
	t.Helper()
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("GET /me status = %d; body=%s", res.Code, res.Body.String())
	}
	var envelope struct {
		Data struct {
			Roles []string `json:"roles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode GET /me: %v", err)
	}
	if fmt.Sprint(envelope.Data.Roles) != fmt.Sprint(want) {
		t.Fatalf("roles = %v, want %v", envelope.Data.Roles, want)
	}
}

func (f *meCacheIntegrationFixture) mutateRole(t *testing.T, method, path, body string) {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status = %d; body=%s", method, path, response.Code, response.Body.String())
	}
}

func TestMeCacheReflectsRoleAssignmentAndRemovalImmediately(t *testing.T) {
	fixture := newMeCacheIntegrationFixture(t)
	fixture.assertRoles(t, []string{"learner"})
	fixture.mutateRole(
		t,
		http.MethodPost,
		"/api/internal-v1/rbac/users/"+roleBindingTestUserID+"/roles",
		`{"role_id":2}`,
	)
	fixture.assertRoles(t, []string{"admin", "learner"})
	fixture.mutateRole(
		t,
		http.MethodDelete,
		"/api/internal-v1/rbac/users/"+roleBindingTestUserID+"/roles/2",
		"",
	)
	fixture.assertRoles(t, []string{"learner"})
}

func TestMeRoleReadFailureDoesNotCachePartialProfile(t *testing.T) {
	fixture := newMeCacheIntegrationFixture(t)
	fixture.roleRepo.listErr = errors.New("role lookup failed")

	response := httptest.NewRecorder()
	fixture.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("GET /me status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	fixture.cache.mu.Lock()
	defer fixture.cache.mu.Unlock()
	if len(fixture.cache.values) != 0 {
		t.Fatalf("cached values = %v, want none", fixture.cache.values)
	}
}
