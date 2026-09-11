package server

import (
	"context"

	"gorm.io/gorm"

	authzapp "mycourse-io-be/internal/authorization/application"
	authzdomain "mycourse-io-be/internal/authorization/domain"
	authzinfra "mycourse-io-be/internal/authorization/infra"
)

func wireAuthorization(
	db *gorm.DB,
	providers ...authzdomain.PolicyProvider,
) error {
	registry, err := authzapp.NewRegistry(providers...)
	if err != nil {
		return err
	}
	grantRepo := authzinfra.NewGormGrantRepository(db)
	grants := authzapp.NewGrantService(registry, grantRepo)
	return grants.SyncActions(context.Background())
}
