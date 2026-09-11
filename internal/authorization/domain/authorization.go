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

type GrantRepository interface {
	UpsertActions(ctx context.Context, actions []ActionDefinition) error
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

var (
	ErrInvalidProvider          = errors.New("invalid authorization policy provider")
	ErrDuplicateProvider        = errors.New("duplicate authorization policy provider")
	ErrDuplicateAction          = errors.New("duplicate authorization action")
	ErrActionCatalogConflict    = errors.New("authorization action catalog conflict")
	ErrUnknownAction            = errors.New("unknown authorization action")
	ErrInvalidGrant             = errors.New("invalid authorization grant")
	ErrGrantManagementForbidden = errors.New("authorization grant management forbidden")
	ErrInvalidPolicyContext     = errors.New("invalid authorization policy context")
)
