package application

import (
	"fmt"
	"sort"
	"strings"

	"mycourse-io-be/internal/authorization/domain"
)

// Registry owns the immutable provider and action catalog used by the decision engine.
type Registry struct {
	providers map[string]domain.PolicyProvider
	actions   map[string]domain.ActionDefinition
}

func NewRegistry(providers ...domain.PolicyProvider) (*Registry, error) {
	registry := &Registry{
		providers: make(map[string]domain.PolicyProvider, len(providers)),
		actions:   make(map[string]domain.ActionDefinition),
	}
	for _, provider := range providers {
		if err := registry.addProvider(provider); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) addProvider(provider domain.PolicyProvider) error {
	if provider == nil {
		return domain.ErrInvalidProvider
	}
	resourceType := strings.TrimSpace(provider.ResourceType())
	if resourceType == "" {
		return domain.ErrInvalidProvider
	}
	if _, exists := r.providers[resourceType]; exists {
		return fmt.Errorf("%w: %s", domain.ErrDuplicateProvider, resourceType)
	}
	actions := provider.Actions()
	if len(actions) == 0 {
		return fmt.Errorf("%w: provider %s has no actions", domain.ErrInvalidProvider, resourceType)
	}
	for _, action := range actions {
		action.Name = strings.TrimSpace(action.Name)
		action.ResourceType = strings.TrimSpace(action.ResourceType)
		action.BoundaryPermission = strings.TrimSpace(action.BoundaryPermission)
		action.Description = strings.TrimSpace(action.Description)
		if action.Name == "" || action.ResourceType != resourceType || action.BoundaryPermission == "" {
			return fmt.Errorf("%w: action %q", domain.ErrInvalidProvider, action.Name)
		}
		if _, exists := r.actions[action.Name]; exists {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateAction, action.Name)
		}
		r.actions[action.Name] = action
	}
	r.providers[resourceType] = provider
	return nil
}

func (r *Registry) Provider(resourceType string) (domain.PolicyProvider, bool) {
	provider, ok := r.providers[strings.TrimSpace(resourceType)]
	return provider, ok
}

func (r *Registry) Action(name string) (domain.ActionDefinition, bool) {
	action, ok := r.actions[strings.TrimSpace(name)]
	return action, ok
}

func (r *Registry) Actions() []domain.ActionDefinition {
	actions := make([]domain.ActionDefinition, 0, len(r.actions))
	for _, action := range r.actions {
		actions = append(actions, action)
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	return actions
}

func (r *Registry) GrantableActions(resourceType string) []string {
	actions := make([]string, 0)
	for _, action := range r.actions {
		if action.ResourceType == resourceType && action.Grantable {
			actions = append(actions, action.Name)
		}
	}
	sort.Strings(actions)
	return actions
}
