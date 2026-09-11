package infra

import (
	"strings"
	"testing"

	"mycourse-io-be/internal/shared/constants"
)

func TestPermissionCodesForUsersSQLUnionsRoleAndDirectPermissions(t *testing.T) {
	for _, fragment := range []string{
		constants.TableRBACUserRoles,
		constants.TableRBACRolePermissions,
		constants.TableRBACUserPermissions,
		"UNION",
		"ORDER BY user_id, permission_name",
	} {
		if !strings.Contains(sqlPermissionCodesForUsers, fragment) {
			t.Fatalf("batch permission SQL missing %q", fragment)
		}
	}
	if got := strings.Count(sqlPermissionCodesForUsers, "IN ?"); got != 2 {
		t.Fatalf("batch permission SQL has %d IN clauses, want 2", got)
	}
}
