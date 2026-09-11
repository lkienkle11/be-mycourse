package server

import (
	"context"
	"errors"
	"reflect"
	"testing"

	rbacdomain "mycourse-io-be/internal/rbac/domain"
)

type roleNameReaderStub struct {
	roles []rbacdomain.Role
	err   error
	calls int
}

func (s *roleNameReaderStub) ListRolesForUser(context.Context, string) ([]rbacdomain.Role, error) {
	s.calls++
	return s.roles, s.err
}

func TestRBACRoleNameReader(t *testing.T) {
	t.Parallel()

	stub := &roleNameReaderStub{roles: []rbacdomain.Role{
		{Name: "admin", Permissions: []rbacdomain.Permission{{PermissionName: "user:read"}}},
		{Name: "instructor"},
	}}
	reader := &rbacRoleNameReader{svc: stub}
	names, err := reader.RoleNamesForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("RoleNamesForUser() err = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"admin", "instructor"}) {
		t.Fatalf("names = %v", names)
	}
	if stub.calls != 1 {
		t.Fatalf("calls = %d, want 1", stub.calls)
	}
}

func TestRBACRoleNameReaderError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("list roles failed")
	stub := &roleNameReaderStub{err: wantErr}
	reader := &rbacRoleNameReader{svc: stub}
	if _, err := reader.RoleNamesForUser(context.Background(), "user-1"); !errors.Is(err, wantErr) {
		t.Fatalf("RoleNamesForUser() err = %v, want %v", err, wantErr)
	}
}

func TestRBACRoleNameReaderEmpty(t *testing.T) {
	t.Parallel()

	stub := &roleNameReaderStub{roles: []rbacdomain.Role{}}
	names, err := (&rbacRoleNameReader{svc: stub}).RoleNamesForUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("RoleNamesForUser() err = %v", err)
	}
	if names == nil || len(names) != 0 {
		t.Fatalf("names = %#v, want non-nil empty", names)
	}
	if stub.calls != 1 {
		t.Fatalf("calls = %d, want 1", stub.calls)
	}
}
