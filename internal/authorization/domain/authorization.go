// Package domain defines transport- and persistence-independent authorization contracts.
package domain

import (
	"context"
	"errors"

	"mycourse-io-be/internal/shared/requestprincipal"
)

type Effect string

const (
	EffectAllow Effect = "ALLOW"
	EffectDeny  Effect = "DENY"
)

type PrincipalType = requestprincipal.Type

const PrincipalTypeUser = requestprincipal.TypeUser

type Principal = requestprincipal.Principal

type ActionDefinition struct {
	Name               string
	ResourceType       string
	BoundaryPermission string
	Description        string
	Grantable          bool
}

type Resource struct {
	Type string
	ID   string
}

// WildcardResourceID is the reserved authorization_role_bindings.resource_id value meaning
// "every resource of this binding's resource_type", scoped to that resource type only. A real
// resource ID is never this literal value, so no collision is possible.
const WildcardResourceID = "*"

type EvaluationContext struct {
	Attributes  map[string]any
	DomainFacts any
}

type AuthorizationRequest struct {
	Principal Principal
	Action    string
	Resource  Resource
	Context   EvaluationContext
}

type DecisionReason string

const (
	ReasonInvalidPrincipal        DecisionReason = "invalid_principal"
	ReasonUnknownAction           DecisionReason = "unknown_action"
	ReasonResourceTypeMismatch    DecisionReason = "resource_type_mismatch"
	ReasonMissingGlobalPermission DecisionReason = "missing_global_permission"
	ReasonProviderDeny            DecisionReason = "provider_deny"
	ReasonExplicitDeny            DecisionReason = "explicit_deny"
	ReasonProviderAllow           DecisionReason = "provider_allow"
	ReasonGrantAllow              DecisionReason = "grant_allow"
	ReasonImplicitDeny            DecisionReason = "implicit_deny"
	ReasonEvaluationError         DecisionReason = "evaluation_error"
)

type Decision struct {
	Effect          Effect
	Reason          DecisionReason
	MatchedGrantIDs []string
}

func (d Decision) Allowed() bool { return d.Effect == EffectAllow }

type PolicyEffect string

const (
	PolicyNeutral PolicyEffect = "NEUTRAL"
	PolicyAllow   PolicyEffect = "ALLOW"
	PolicyDeny    PolicyEffect = "DENY"
)

type PolicyDecision struct {
	Effect PolicyEffect
	Reason DecisionReason
}

type GrantManagementRequest struct {
	Issuer             Principal
	TargetPrincipalIDs []string
	Resource           Resource
	Context            EvaluationContext
}

type PolicyProvider interface {
	ResourceType() string
	Actions() []ActionDefinition
	Evaluate(AuthorizationRequest) (PolicyDecision, error)
	CanManageGrants(GrantManagementRequest) (bool, error)
}

type ConditionOperator string

const (
	ConditionStringEquals  ConditionOperator = "StringEquals"
	ConditionBoolEquals    ConditionOperator = "BoolEquals"
	ConditionNumericEquals ConditionOperator = "NumericEquals"
	ConditionDateBefore    ConditionOperator = "DateBefore"
	ConditionDateAfter     ConditionOperator = "DateAfter"
)

type Condition struct {
	Operator ConditionOperator
	Key      string
	Values   []any
}

type Grant struct {
	ID              string
	PrincipalUserID string
	ActionName      string
	ResourceType    string
	ResourceID      string
	Effect          Effect
	Conditions      []Condition
	ValidFrom       *int64
	ExpiresAt       *int64
	GrantedByUserID string
	CreatedAt       int64
	RevokedAt       *int64
}

func (g Grant) ActiveAt(now int64) bool {
	if g.RevokedAt != nil {
		return false
	}
	if g.ValidFrom != nil && *g.ValidFrom > now {
		return false
	}
	return g.ExpiresAt == nil || *g.ExpiresAt > now
}

type GrantQuery struct {
	PrincipalUserIDs []string
	ActionNames      []string
	Resource         Resource
	Effects          []Effect
}

type GrantReplacement struct {
	PrincipalUserIDs []string
	ManagedActions   []string
	Actions          []string
	Resource         Resource
	Effect           Effect
	Conditions       []Condition
	ValidFrom        *int64
	ExpiresAt        *int64
	GrantedByUserID  string
}

// RoleActionDefinition declares that roleName covers actionName for resourceType — one row of
// authorization_role_actions. A resource type's own package declares these (e.g. Course's
// OWNER/EDITOR mapping) as static, code-declared seed data, analogous to how a PolicyProvider
// declares its action catalog; internal/authorization itself never hardcodes a role name.
type RoleActionDefinition struct {
	RoleName     string
	ResourceType string
	ActionName   string
}

type GrantRepository interface {
	UpsertActions(ctx context.Context, actions []ActionDefinition) error
	// SyncRoleActions idempotently ensures every (RoleName, ResourceType, ActionName) tuple in
	// roleActions exists in authorization_role_actions. It only adds; it never removes a tuple
	// that is no longer declared, matching UpsertActions' append-only stance on the action
	// catalog.
	SyncRoleActions(ctx context.Context, roleActions []RoleActionDefinition) error
	// ListActive may include grants synthesized from an active resource-scoped role
	// binding (a principal holding a named role on a resource, expanded to whatever
	// actions that role currently covers) in addition to directly stored ones. Every
	// returned Grant is a plain Grant regardless of source; callers cannot and do not
	// need to distinguish the two.
	ListActive(ctx context.Context, query GrantQuery, now int64) ([]Grant, error)
	Create(ctx context.Context, grant *Grant) error
	Revoke(ctx context.Context, query GrantQuery, revokedAt int64) error
	ReplaceMany(ctx context.Context, replacement GrantReplacement, now int64) error
}

// RoleBindingAssignment assigns roleName to every principal in PrincipalUserIDs for Resource
// (or every resource of Resource.Type, when Resource.ID is WildcardResourceID), in one call —
// never one call per principal (see .ai/skills/logic-n-1-optimize).
type RoleBindingAssignment struct {
	Issuer           Principal
	PrincipalUserIDs []string
	RoleName         string
	Resource         Resource
	Context          EvaluationContext
}

// RoleBindingRevocation ends roleName for every principal in PrincipalUserIDs on Resource, in
// one call.
type RoleBindingRevocation struct {
	Issuer           Principal
	PrincipalUserIDs []string
	RoleName         string
	Resource         Resource
	Context          EvaluationContext
}

// RoleBindingQuery selects active role_bindings rows, to revoke or to list.
type RoleBindingQuery struct {
	PrincipalUserIDs []string
	RoleName         string
	Resource         Resource
}

// RoleBinding is one active authorization_role_bindings row, returned by ListRoleBindings for
// display purposes (e.g. a resource's collaborator list, or an exclusion set for a candidate
// picker). It is never used to gate an authorization decision — only
// GrantRepository.ListActive's role-expansion does that.
type RoleBinding struct {
	ID              string
	PrincipalUserID string
	RoleName        string
	Resource        Resource
	GrantedByUserID string
	CreatedAt       int64
}

// RoleBindingRepository is the write path for resource-scoped role bindings
// (authorization_role_bindings), plus the one sanctioned read path for callers that need to
// list or display bindings rather than gate a decision (ListRoleBindings). Callers outside
// internal/authorization must not query authorization_role_bindings directly; an authorization
// decision itself must still go through GrantRepository.ListActive's role-expansion, never
// through ListRoleBindings.
type RoleBindingRepository interface {
	// AssignRoleBindings inserts one active binding per principal in principalUserIDs, in a
	// single batched write.
	AssignRoleBindings(ctx context.Context, principalUserIDs []string, roleName string, resource Resource, grantedByUserID string, now int64) error
	// RevokeRoleBindings marks every active binding matching query as revoked, in a single
	// batched write. Matching zero rows (e.g. an already-revoked binding) is not an error.
	RevokeRoleBindings(ctx context.Context, query RoleBindingQuery, revokedAt int64) error
	// ListRoleBindings returns every active binding matching query, for display purposes only.
	ListRoleBindings(ctx context.Context, query RoleBindingQuery) ([]RoleBinding, error)
}

var (
	ErrInvalidProvider          = errors.New("invalid authorization policy provider")
	ErrDuplicateProvider        = errors.New("duplicate authorization policy provider")
	ErrDuplicateAction          = errors.New("duplicate authorization action")
	ErrActionCatalogConflict    = errors.New("authorization action catalog conflict")
	ErrUnknownAction            = errors.New("unknown authorization action")
	ErrInvalidGrant             = errors.New("invalid authorization grant")
	ErrInvalidRoleBinding       = errors.New("invalid authorization role binding")
	ErrGrantManagementForbidden = errors.New("authorization grant management forbidden")
	ErrInvalidPolicyContext     = errors.New("invalid authorization policy context")
)
