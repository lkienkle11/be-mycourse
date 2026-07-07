package server

import (
	"context"

	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authdomain "mycourse-io-be/internal/auth/domain"
	authinfra "mycourse-io-be/internal/auth/infra"
	rbacinfra "mycourse-io-be/internal/rbac/infra"
)

// authEmailConfirmer atomically persists confirm fields and assigns the learner role.
type authEmailConfirmer struct {
	db           *gorm.DB
	userRepo     *authinfra.GormUserRepository
	userRoleRepo *rbacinfra.GormUserRoleRepository
	roleRepo     *rbacinfra.GormRoleRepository
}

func newAuthEmailConfirmer(
	db *gorm.DB,
	userRepo *authinfra.GormUserRepository,
	userRoleRepo *rbacinfra.GormUserRoleRepository,
	roleRepo *rbacinfra.GormRoleRepository,
) authapp.EmailConfirmer {
	return &authEmailConfirmer{
		db: db, userRepo: userRepo, userRoleRepo: userRoleRepo, roleRepo: roleRepo,
	}
}

func (c *authEmailConfirmer) ConfirmEmailWithLearnerRole(ctx context.Context, user *authdomain.User) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := c.userRepo.SaveWithDB(ctx, tx, user); err != nil {
			return err
		}
		return assignLearnerRoleWithDB(ctx, tx, c.userRoleRepo, c.roleRepo, user.ID)
	})
}
