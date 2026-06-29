package gormx

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
)

// UserIDSetByRoleNames returns user IDs from userIDs that have at least one of the given role names.
func UserIDSetByRoleNames(
	ctx context.Context,
	db *gorm.DB,
	userIDs []string,
	roleNames []string,
) (map[string]struct{}, error) {
	if len(userIDs) == 0 || len(roleNames) == 0 {
		return map[string]struct{}{}, nil
	}
	var matched []string
	err := db.WithContext(ctx).Raw(fmt.Sprintf(`
SELECT DISTINCT ur.user_id::text
FROM %s ur
INNER JOIN %s ro ON ro.id = ur.role_id
WHERE ur.user_id IN ? AND ro.name IN ?`, constants.TableRBACUserRoles, constants.TableRBACRoles),
		userIDs, roleNames).Scan(&matched).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(matched))
	for _, id := range matched {
		out[id] = struct{}{}
	}
	return out, nil
}
