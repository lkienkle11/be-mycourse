package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"mycourse-io-be/internal/rbac/domain"
)

type batchUserPermissionRepository struct {
	result    map[string]map[string]struct{}
	err       error
	calls     int
	requested []string
}

func (*batchUserPermissionRepository) ListPermissionsForUser(context.Context, string) ([]domain.Permission, error) {
	return nil, nil
}

func (*batchUserPermissionRepository) PermissionCodesForUser(context.Context, string) (map[string]struct{}, error) {
	return nil, nil
}

func (r *batchUserPermissionRepository) PermissionCodesForUsers(
	_ context.Context,
	userIDs []string,
) (map[string]map[string]struct{}, error) {
	r.calls++
	r.requested = append([]string(nil), userIDs...)
	return r.result, r.err
}

func (*batchUserPermissionRepository) AssignPermission(context.Context, string, string) error {
	return nil
}

func (*batchUserPermissionRepository) AssignPermissionByName(context.Context, string, string) error {
	return nil
}

func (*batchUserPermissionRepository) RemovePermission(context.Context, string, string) error {
	return nil
}

func TestPermissionCodesForUsersNormalizesAndKeepsEmptyEntries(t *testing.T) {
	repo := &batchUserPermissionRepository{result: map[string]map[string]struct{}{
		"u1": {"course:update": {}, "course:read": {}},
	}}
	service := NewRBACService(nil, nil, nil, repo)

	got, err := service.PermissionCodesForUsers(context.Background(), []string{" u1 ", "u2", "u1", " "})
	if err != nil {
		t.Fatalf("PermissionCodesForUsers: %v", err)
	}
	if want := []string{"u1", "u2"}; !reflect.DeepEqual(repo.requested, want) {
		t.Fatalf("repository IDs = %#v, want %#v", repo.requested, want)
	}
	if len(got["u1"]) != 2 {
		t.Fatalf("u1 permissions = %#v, want role/direct union", got["u1"])
	}
	if got["u2"] == nil || len(got["u2"]) != 0 {
		t.Fatalf("u2 permissions = %#v, want initialized empty set", got["u2"])
	}
}

func TestPermissionCodesForUsersEmptyAndRepositoryError(t *testing.T) {
	repo := &batchUserPermissionRepository{}
	service := NewRBACService(nil, nil, nil, repo)
	got, err := service.PermissionCodesForUsers(context.Background(), []string{"", " "})
	if err != nil || len(got) != 0 || repo.calls != 0 {
		t.Fatalf("empty lookup = %#v, %v, calls=%d; want empty without query", got, err, repo.calls)
	}

	repo.err = errors.New("database failed")
	got, err = service.PermissionCodesForUsers(context.Background(), []string{"u1"})
	if err == nil || got != nil {
		t.Fatalf("failed lookup = %#v, %v; want nil and error", got, err)
	}
}

var _ domain.UserPermissionRepository = (*batchUserPermissionRepository)(nil)
