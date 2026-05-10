// Package errfuncrbac holds RBAC-related error helper functions (Rule 19).
package errfuncrbac

import (
	"fmt"

	pkgerrors "mycourse-io-be/pkg/errors"
)

// WrapRBACUnknownPermissionID annotates ErrRBACUnknownPermissionID with the offending id (for errors.Is on the base sentinel).
func WrapRBACUnknownPermissionID(permissionID string) error {
	return fmt.Errorf("%w: %q", pkgerrors.ErrRBACUnknownPermissionID, permissionID)
}
