package application

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/redis/go-redis/v9"

	"mycourse-io-be/internal/auth/domain"
)

type roleTestPermissionReader struct {
	codes map[string]struct{}
	err   error
}

func (r roleTestPermissionReader) PermissionCodesForUser(string) (map[string]struct{}, error) {
	return r.codes, r.err
}

type roleTestNameReader struct {
	names []string
	err   error
	calls int
}

func (r *roleTestNameReader) RoleNamesForUser(context.Context, string) ([]string, error) {
	r.calls++
	return r.names, r.err
}

func TestSortRolesByRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "ranked", input: []string{"learner", "sysadmin", "instructor", "admin"}, want: []string{"sysadmin", "admin", "instructor", "learner"}},
		{name: "unknown stable", input: []string{"reviewer", "learner", "editor", "admin"}, want: []string{"admin", "learner", "reviewer", "editor"}},
		{name: "empty", input: []string{}, want: []string{}},
		{name: "nil", input: nil, want: []string{}},
		{name: "single", input: []string{"instructor"}, want: []string{"instructor"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original := append([]string(nil), tc.input...)
			got := sortRolesByRank(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("sortRolesByRank() = %v, want %v", got, tc.want)
			}
			unchanged := len(tc.input) == len(original)
			if unchanged {
				for i := range tc.input {
					unchanged = unchanged && tc.input[i] == original[i]
				}
			}
			if !unchanged {
				t.Fatalf("sortRolesByRank() mutated input: got %v, original %v", tc.input, original)
			}
			if got == nil {
				t.Fatal("sortRolesByRank() returned nil")
			}
		})
	}
}

func TestLoadMeProfileIncludesDisplayRoles(t *testing.T) {
	t.Parallel()

	reader := &roleTestNameReader{names: []string{"learner", "admin"}}
	svc := &AuthService{
		permReader:     roleTestPermissionReader{codes: map[string]struct{}{"course:read": {}}},
		roleNameReader: reader,
	}
	user := &domain.User{ID: "user-1", Email: "user@example.com"}

	profile, err := svc.loadMeProfile(context.Background(), user)
	if err != nil {
		t.Fatalf("loadMeProfile() err = %v", err)
	}
	if !reflect.DeepEqual(profile.Roles, []string{"admin", "learner"}) {
		t.Fatalf("roles = %v", profile.Roles)
	}
	if !reflect.DeepEqual(profile.Permissions, []string{"course:read"}) {
		t.Fatalf("permissions = %v", profile.Permissions)
	}
	if reader.calls != 1 {
		t.Fatalf("role reader calls = %d, want 1", reader.calls)
	}
}

func TestLoadMeProfileKeepsNoRolesNonNil(t *testing.T) {
	t.Parallel()

	svc := &AuthService{roleNameReader: &roleTestNameReader{}}
	profile, err := svc.loadMeProfile(context.Background(), &domain.User{ID: "user-1"})
	if err != nil {
		t.Fatalf("loadMeProfile() err = %v", err)
	}
	if profile.Roles == nil || len(profile.Roles) != 0 {
		t.Fatalf("roles = %#v, want non-nil empty", profile.Roles)
	}
}

func TestLoadMeProfilePropagatesRoleReaderError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("role lookup failed")
	svc := &AuthService{roleNameReader: &roleTestNameReader{err: wantErr}}
	if _, err := svc.loadMeProfile(context.Background(), &domain.User{ID: "user-1"}); !errors.Is(err, wantErr) {
		t.Fatalf("loadMeProfile() err = %v, want %v", err, wantErr)
	}
}

func TestWarmMeCacheLoadsRolesBeforeBestEffortWrite(t *testing.T) {
	t.Parallel()

	reader := &roleTestNameReader{names: []string{"learner"}}
	rdb := redis.NewClient(&redis.Options{
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("cache unavailable")
		},
	})
	t.Cleanup(func() { _ = rdb.Close() })
	svc := &AuthService{redis: rdb, roleNameReader: reader}

	svc.warmMeCache(context.Background(), &domain.User{ID: "user-1"})
	if reader.calls != 1 {
		t.Fatalf("role reader calls = %d, want 1", reader.calls)
	}
}
