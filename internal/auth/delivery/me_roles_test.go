package delivery

import (
	"encoding/json"
	"reflect"
	"testing"

	"mycourse-io-be/internal/auth/domain"
)

func TestToMeResponseRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		roles []string
		want  []string
	}{
		{name: "nil becomes empty", roles: nil, want: []string{}},
		{name: "ordered values preserved", roles: []string{"admin", "instructor"}, want: []string{"admin", "instructor"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := toMeResponse(&domain.MeProfile{UserID: "user-1", Roles: tc.roles})
			if !reflect.DeepEqual(response.Roles, tc.want) {
				t.Fatalf("roles = %v, want %v", response.Roles, tc.want)
			}
			data, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("json.Marshal() err = %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() err = %v", err)
			}
			if _, ok := decoded["roles"].([]any); !ok {
				t.Fatalf("roles JSON = %#v, want array", decoded["roles"])
			}
		})
	}
}
