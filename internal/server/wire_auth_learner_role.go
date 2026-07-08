package server

import (
	"context"

	"gorm.io/gorm"

	rbacinfra "mycourse-io-be/internal/rbac/infra"
)

// assignLearnerRoleWithDB looks up the learner role and assigns it inside an open transaction.
func assignLearnerRoleWithDB(
	ctx context.Context,
	tx *gorm.DB,
	userRoleRepo *rbacinfra.GormUserRoleRepository,
	roleRepo *rbacinfra.GormRoleRepository,
	userID string,
) error {
	role, err := roleRepo.GetByName(ctx, "learner")
	if err != nil {
		return err
	}
	return userRoleRepo.AssignRoleWithDB(ctx, tx, userID, role.ID)
}
