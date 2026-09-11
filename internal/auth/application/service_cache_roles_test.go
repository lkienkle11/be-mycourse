package application

import (
	"encoding/json"
	"reflect"
	"testing"

	"mycourse-io-be/internal/auth/domain"
)

func TestDecodeCachedMeRolesCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		user string
		ok   bool
		want []string
	}{
		{name: "legacy missing roles", data: `{"UserID":"user-1"}`, user: "user-1", ok: false},
		{name: "explicit empty roles", data: `{"UserID":"user-1","Roles":[]}`, user: "user-1", ok: true, want: []string{}},
		{name: "populated roles", data: `{"UserID":"user-1","Roles":["admin"]}`, user: "user-1", ok: true, want: []string{"admin"}},
		{name: "wrong user", data: `{"UserID":"user-2","Roles":[]}`, user: "user-1", ok: false},
		{name: "malformed", data: `{`, user: "user-1", ok: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			profile, ok := decodeCachedMe([]byte(tc.data), tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				if profile != nil {
					t.Fatalf("profile = %#v, want nil", profile)
				}
				return
			}
			if !reflect.DeepEqual(profile.Roles, tc.want) {
				t.Fatalf("roles = %v, want %v", profile.Roles, tc.want)
			}
		})
	}
}

func TestBuildMeProfileSerializesNoRolesAsArray(t *testing.T) {
	t.Parallel()

	profile := buildMeProfile(&domain.User{ID: "user-1"}, nil, nil)
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("json.Marshal() err = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() err = %v", err)
	}
	roles, ok := decoded["Roles"].([]any)
	if !ok || len(roles) != 0 {
		t.Fatalf("Roles = %#v, want empty JSON array", decoded["Roles"])
	}
}
