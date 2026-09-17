package server

import (
	"context"

	"gorm.io/gorm"

	authzapp "mycourse-io-be/internal/authorization/application"
	authzdomain "mycourse-io-be/internal/authorization/domain"
	authzinfra "mycourse-io-be/internal/authorization/infra"
)

// AuthorizationServices holds the runtime pieces of internal/authorization that a resource
// type's own wiring (e.g. wireCourse) needs: an Authorizer to gate access checks, and a
// RoleBindingService to assign/revoke resource-scoped role bindings.
type AuthorizationServices struct {
	Authorizer         *authzapp.Authorizer
	RoleBindingService *authzapp.RoleBindingService
}

// wireAuthorization registers every PolicyProvider, syncs their action catalog, seeds any
// declared role-action tuples (roleActions — see e.g. courseapp.RoleActions), and returns the
// runtime services built on top.
func wireAuthorization(
	db *gorm.DB,
	roleActions []authzdomain.RoleActionDefinition,
	providers ...authzdomain.PolicyProvider,
) (*AuthorizationServices, error) {
	registry, err := authzapp.NewRegistry(providers...)
	if err != nil {
		return nil, err
	}
	grantRepo := authzinfra.NewGormGrantRepository(db)
	grants := authzapp.NewGrantService(registry, grantRepo)
	if err := grants.SyncActions(context.Background()); err != nil {
		return nil, err
	}
	if err := grantRepo.SyncRoleActions(context.Background(), roleActions); err != nil {
		return nil, err
	}
	return &AuthorizationServices{
		Authorizer:         authzapp.NewAuthorizer(registry, grantRepo),
		RoleBindingService: authzapp.NewRoleBindingService(registry, grantRepo),
	}, nil
}
