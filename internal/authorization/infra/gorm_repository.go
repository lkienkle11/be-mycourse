package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mycourse-io-be/internal/authorization/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/uuidx"
)

const grantInsertBatchSize = 100

type GormGrantRepository struct {
	db *gorm.DB
}

func NewGormGrantRepository(db *gorm.DB) *GormGrantRepository {
	return &GormGrantRepository{db: db}
}

type actionRow struct {
	ActionName         string `gorm:"column:action_name;primaryKey"`
	ResourceType       string `gorm:"column:resource_type;primaryKey"`
	BoundaryPermission string `gorm:"column:boundary_permission;not null"`
	Description        string `gorm:"column:description;not null"`
	CreatedAt          int64  `gorm:"column:created_at;not null"`
	UpdatedAt          int64  `gorm:"column:updated_at;not null"`
}

func (actionRow) TableName() string { return constants.TableAuthorizationActions }

type grantRow struct {
	ID              string  `gorm:"column:id;primaryKey"`
	PrincipalUserID string  `gorm:"column:principal_user_id;not null"`
	ActionName      string  `gorm:"column:action_name;not null"`
	ResourceType    string  `gorm:"column:resource_type;not null"`
	ResourceID      string  `gorm:"column:resource_id;not null"`
	Effect          string  `gorm:"column:effect;not null"`
	Conditions      []byte  `gorm:"column:conditions;type:jsonb;not null"`
	ValidFrom       *int64  `gorm:"column:valid_from"`
	ExpiresAt       *int64  `gorm:"column:expires_at"`
	GrantedByUserID *string `gorm:"column:granted_by_user_id"`
	CreatedAt       int64   `gorm:"column:created_at;not null"`
	RevokedAt       *int64  `gorm:"column:revoked_at"`
}

func (grantRow) TableName() string { return constants.TableAuthorizationGrants }

// roleActionRow is the role definition: which actions a role name currently covers, for a
// resource type. Redefining it (insert/delete a row) takes effect immediately for every
// existing roleBindingRow referencing that role, since bindings never store a copy of it.
type roleActionRow struct {
	RoleName     string `gorm:"column:role_name;primaryKey"`
	ResourceType string `gorm:"column:resource_type;primaryKey"`
	ActionName   string `gorm:"column:action_name;primaryKey"`
	CreatedAt    int64  `gorm:"column:created_at;not null"`
}

func (roleActionRow) TableName() string { return constants.TableAuthorizationRoleActions }

// roleBindingRow is the role assignment: which principal holds which role on which specific
// resource instance. Its actual actions are resolved by joining RoleName against
// roleActionRow at decision time (roleExpandedGrants), never stored here.
type roleBindingRow struct {
	ID              string  `gorm:"column:id;primaryKey"`
	PrincipalUserID string  `gorm:"column:principal_user_id;not null"`
	RoleName        string  `gorm:"column:role_name;not null"`
	ResourceType    string  `gorm:"column:resource_type;not null"`
	ResourceID      string  `gorm:"column:resource_id;not null"`
	GrantedByUserID *string `gorm:"column:granted_by_user_id"`
	CreatedAt       int64   `gorm:"column:created_at;not null"`
	RevokedAt       *int64  `gorm:"column:revoked_at"`
}

func (roleBindingRow) TableName() string { return constants.TableAuthorizationRoleBindings }

type conditionPayload struct {
	Operator string `json:"operator"`
	Key      string `json:"key"`
	Values   []any  `json:"values"`
}

// dbForContext returns the transaction a caller joined via gormx.WithTx, when present, so a
// caller from a different bounded context (e.g. internal/course) can make this repository's
// writes part of its own transaction instead of a separately committed one. Falls back to this
// repository's own *gorm.DB otherwise.
func (r *GormGrantRepository) dbForContext(ctx context.Context) *gorm.DB {
	return gormx.DBFromContext(ctx, r.db)
}

func (r *GormGrantRepository) UpsertActions(ctx context.Context, actions []domain.ActionDefinition) error {
	if len(actions) == 0 {
		return nil
	}
	now := timex.NowUnix()
	actionNames := make([]string, len(actions))
	rows := make([]actionRow, len(actions))
	for i, action := range actions {
		actionNames[i] = action.Name
		rows[i] = actionRow{
			ActionName: action.Name, ResourceType: action.ResourceType,
			BoundaryPermission: action.BoundaryPermission, Description: action.Description,
			CreatedAt: now, UpdatedAt: now,
		}
	}
	db := r.dbForContext(ctx)
	var existing []actionRow
	if err := db.Where("action_name IN ?", actionNames).Find(&existing).Error; err != nil {
		return err
	}
	if err := validatePersistedActionRows(actions, existing); err != nil {
		return err
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "action_name"}, {Name: "resource_type"}},
		DoUpdates: clause.Assignments(map[string]any{
			"boundary_permission": gorm.Expr("EXCLUDED.boundary_permission"),
			"description":         gorm.Expr("EXCLUDED.description"),
			"updated_at":          now,
		}),
	}).CreateInBatches(rows, grantInsertBatchSize).Error
}

// SyncRoleActions idempotently inserts any (role_name, resource_type, action_name) tuple in
// roleActions that does not already exist. It never removes an existing tuple, matching
// UpsertActions' append-only stance on the action catalog; the composite primary key makes a
// duplicate insert attempt a no-op rather than an error.
func (r *GormGrantRepository) SyncRoleActions(ctx context.Context, roleActions []domain.RoleActionDefinition) error {
	if len(roleActions) == 0 {
		return nil
	}
	now := timex.NowUnix()
	rows := make([]roleActionRow, len(roleActions))
	for i, roleAction := range roleActions {
		rows[i] = roleActionRow{
			RoleName: roleAction.RoleName, ResourceType: roleAction.ResourceType,
			ActionName: roleAction.ActionName, CreatedAt: now,
		}
	}
	return r.dbForContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, grantInsertBatchSize).Error
}

func validatePersistedActionRows(actions []domain.ActionDefinition, existing []actionRow) error {
	expectedResources := make(map[string]string, len(actions))
	for _, action := range actions {
		expectedResources[action.Name] = action.ResourceType
	}
	for _, row := range existing {
		if expected, ok := expectedResources[row.ActionName]; ok && expected != row.ResourceType {
			return fmt.Errorf("%w: action %s is persisted for resource %s, provider registered %s",
				domain.ErrActionCatalogConflict, row.ActionName, row.ResourceType, expected)
		}
	}
	return nil
}

func (r *GormGrantRepository) ListActive(ctx context.Context, query domain.GrantQuery, now int64) ([]domain.Grant, error) {
	db := r.dbForContext(ctx)
	db = db.Model(&grantRow{}).
		Where("revoked_at IS NULL").
		Where("(valid_from IS NULL OR valid_from <= ?)", now).
		Where("(expires_at IS NULL OR expires_at > ?)", now)
	if len(query.PrincipalUserIDs) > 0 {
		db = db.Where("principal_user_id IN ?", query.PrincipalUserIDs)
	}
	if len(query.ActionNames) > 0 {
		db = db.Where("action_name IN ?", query.ActionNames)
	}
	if query.Resource.Type != "" {
		db = db.Where("resource_type = ?", query.Resource.Type)
	}
	if query.Resource.ID != "" {
		db = db.Where("resource_id = ?", query.Resource.ID)
	}
	if len(query.Effects) > 0 {
		effects := make([]string, len(query.Effects))
		for i, effect := range query.Effects {
			effects[i] = string(effect)
		}
		db = db.Where("effect IN ?", effects)
	}
	var rows []grantRow
	if err := db.Order("principal_user_id ASC, action_name ASC, effect ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	directGrants := make([]domain.Grant, len(rows))
	for i := range rows {
		grant, err := rowToGrant(&rows[i])
		if err != nil {
			return nil, err
		}
		directGrants[i] = grant
	}
	roleGrants, err := r.roleExpandedGrants(ctx, query)
	if err != nil {
		return nil, err
	}
	return mergeGrants(directGrants, roleGrants), nil
}

// roleExpandedGrantRow is one (role binding, covered action) pair produced by joining an
// active authorization_role_bindings row to every authorization_role_actions row that
// shares its role_name and resource_type.
type roleExpandedGrantRow struct {
	BindingID       string  `gorm:"column:id"`
	PrincipalUserID string  `gorm:"column:principal_user_id"`
	ResourceType    string  `gorm:"column:resource_type"`
	ResourceID      string  `gorm:"column:resource_id"`
	ActionName      string  `gorm:"column:action_name"`
	GrantedByUserID *string `gorm:"column:granted_by_user_id"`
	CreatedAt       int64   `gorm:"column:created_at"`
}

// roleBindingResourceClause returns the WHERE fragment and bind args matching a role binding
// against a requested resource ID: either the exact resource ID, or a binding scoped to
// domain.WildcardResourceID (every resource of the binding's own resource_type). Extracted as
// its own function so the matching rule is unit-testable without a database (see
// gorm_repository_test.go); roleExpandedGrants is the only caller.
func roleBindingResourceClause(resourceID string) (string, []any) {
	return "(rb.resource_id = ? OR rb.resource_id = ?)", []any{resourceID, domain.WildcardResourceID}
}

// roleExpandedGrants resolves active resource-scoped role bindings into synthesized ALLOW
// grants, one per action the binding's role currently covers for its resource type. It never
// sets ValidFrom/ExpiresAt/Conditions (role bindings do not support them) and is always
// ALLOW-only by construction. Unlike the direct-grant query, there is no now-based validity
// window to check: a role binding is only ever active or revoked (revoked_at IS NULL below).
//
// roleBindingResourceClause lets a wildcard binding match every resource of its own
// resource_type; it can never leak into another resource_type because the resource_type filter
// below is a separate, unconditional AND clause whenever query.Resource.Type is set.
func (r *GormGrantRepository) roleExpandedGrants(ctx context.Context, query domain.GrantQuery) ([]domain.Grant, error) {
	db := r.dbForContext(ctx)
	db = db.Table(roleBindingRow{}.TableName() + " AS rb").
		Select("rb.id AS id, rb.principal_user_id AS principal_user_id, rb.resource_type AS resource_type, " +
			"rb.resource_id AS resource_id, rb.granted_by_user_id AS granted_by_user_id, rb.created_at AS created_at, " +
			"ra.action_name AS action_name").
		Joins("JOIN " + roleActionRow{}.TableName() + " AS ra " +
			"ON ra.role_name = rb.role_name AND ra.resource_type = rb.resource_type").
		Where("rb.revoked_at IS NULL")
	if len(query.PrincipalUserIDs) > 0 {
		db = db.Where("rb.principal_user_id IN ?", query.PrincipalUserIDs)
	}
	if query.Resource.Type != "" {
		db = db.Where("rb.resource_type = ?", query.Resource.Type)
	}
	if query.Resource.ID != "" {
		clause, args := roleBindingResourceClause(query.Resource.ID)
		db = db.Where(clause, args...)
	}
	if len(query.ActionNames) > 0 {
		db = db.Where("ra.action_name IN ?", query.ActionNames)
	}
	var rows []roleExpandedGrantRow
	if err := db.Order("rb.principal_user_id ASC, ra.action_name ASC, rb.id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	grants := make([]domain.Grant, len(rows))
	for i := range rows {
		grants[i] = roleExpandedRowToGrant(&rows[i])
	}
	return grants, nil
}

func roleExpandedRowToGrant(row *roleExpandedGrantRow) domain.Grant {
	return domain.Grant{
		ID:              "role:" + row.BindingID + ":" + row.ActionName,
		PrincipalUserID: row.PrincipalUserID,
		ActionName:      row.ActionName,
		ResourceType:    row.ResourceType,
		ResourceID:      row.ResourceID,
		Effect:          domain.EffectAllow,
		GrantedByUserID: stringValue(row.GrantedByUserID),
		CreatedAt:       row.CreatedAt,
	}
}

// AssignRoleBindings inserts one active authorization_role_bindings row per principal in
// principalUserIDs, in a single batched insert (never one query per principal).
func (r *GormGrantRepository) AssignRoleBindings(
	ctx context.Context,
	principalUserIDs []string,
	roleName string,
	resource domain.Resource,
	grantedByUserID string,
	now int64,
) error {
	if len(principalUserIDs) == 0 {
		return nil
	}
	rows := make([]roleBindingRow, 0, len(principalUserIDs))
	for _, principalID := range principalUserIDs {
		id, err := uuidx.NewV7()
		if err != nil {
			return err
		}
		rows = append(rows, roleBindingRow{
			ID: id, PrincipalUserID: principalID, RoleName: roleName,
			ResourceType: resource.Type, ResourceID: resource.ID,
			GrantedByUserID: stringPointer(grantedByUserID), CreatedAt: now,
		})
	}
	return r.dbForContext(ctx).CreateInBatches(rows, grantInsertBatchSize).Error
}

// RevokeRoleBindings marks every active binding matching query as revoked, in a single batched
// update. Matching zero rows (e.g. a binding that is already revoked, or was never active) is
// not an error. Unlike ListRoleBindings/roleExpandedGrants, a query.Resource.ID of a concrete
// resource intentionally does NOT also match a domain.WildcardResourceID binding here: revoking
// one resource's access must never silently revoke a wildcard binding's access to every other
// resource of that type. Revoking a wildcard binding itself requires querying with
// Resource.ID == domain.WildcardResourceID.
func (r *GormGrantRepository) RevokeRoleBindings(ctx context.Context, query domain.RoleBindingQuery, revokedAt int64) error {
	db := r.dbForContext(ctx).Model(&roleBindingRow{}).Where("revoked_at IS NULL")
	if len(query.PrincipalUserIDs) > 0 {
		db = db.Where("principal_user_id IN ?", query.PrincipalUserIDs)
	}
	if query.RoleName != "" {
		db = db.Where("role_name = ?", query.RoleName)
	}
	if query.Resource.Type != "" {
		db = db.Where("resource_type = ?", query.Resource.Type)
	}
	if query.Resource.ID != "" {
		db = db.Where("resource_id = ?", query.Resource.ID)
	}
	return db.Update("revoked_at", revokedAt).Error
}

// ListRoleBindings returns every active binding matching query, ordered by CreatedAt then ID,
// for callers that need to list or display bindings (e.g. a resource's collaborators) rather
// than gate a decision. This is the one sanctioned read path for authorization_role_bindings
// outside GrantRepository.ListActive's role-expansion. A binding scoped to
// domain.WildcardResourceID matches a concrete query.Resource.ID here too, mirroring
// roleExpandedGrants' matching rule — a wildcard-bound principal must still show up when
// listing (e.g. an EDITOR-of-every-course) collaborators for one specific resource.
func (r *GormGrantRepository) ListRoleBindings(ctx context.Context, query domain.RoleBindingQuery) ([]domain.RoleBinding, error) {
	db := r.dbForContext(ctx).Table(roleBindingRow{}.TableName() + " AS rb").Where("rb.revoked_at IS NULL")
	if len(query.PrincipalUserIDs) > 0 {
		db = db.Where("rb.principal_user_id IN ?", query.PrincipalUserIDs)
	}
	if query.RoleName != "" {
		db = db.Where("rb.role_name = ?", query.RoleName)
	}
	if query.Resource.Type != "" {
		db = db.Where("rb.resource_type = ?", query.Resource.Type)
	}
	if query.Resource.ID != "" {
		clause, args := roleBindingResourceClause(query.Resource.ID)
		db = db.Where(clause, args...)
	}
	var rows []roleBindingRow
	if err := db.Order("rb.created_at ASC, rb.id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.RoleBinding, len(rows))
	for i := range rows {
		out[i] = domain.RoleBinding{
			ID: rows[i].ID, PrincipalUserID: rows[i].PrincipalUserID, RoleName: rows[i].RoleName,
			Resource:        domain.Resource{Type: rows[i].ResourceType, ID: rows[i].ResourceID},
			GrantedByUserID: stringValue(rows[i].GrantedByUserID), CreatedAt: rows[i].CreatedAt,
		}
	}
	return out, nil
}

var _ domain.RoleBindingRepository = (*GormGrantRepository)(nil)

// mergeGrants combines directly stored grants with role-expanded ones into one
// deterministically ordered slice, matching ListActive's pre-existing sort order. No
// deduplication is performed: matchingGrantIDs already tolerates multiple ALLOW entries for
// the same tuple, and a duplicate-looking ALLOW from two different sources remains
// individually traceable via its ID.
func mergeGrants(direct, roleExpanded []domain.Grant) []domain.Grant {
	merged := make([]domain.Grant, 0, len(direct)+len(roleExpanded))
	merged = append(merged, direct...)
	merged = append(merged, roleExpanded...)
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].PrincipalUserID != merged[j].PrincipalUserID {
			return merged[i].PrincipalUserID < merged[j].PrincipalUserID
		}
		if merged[i].ActionName != merged[j].ActionName {
			return merged[i].ActionName < merged[j].ActionName
		}
		if merged[i].Effect != merged[j].Effect {
			return merged[i].Effect < merged[j].Effect
		}
		return merged[i].ID < merged[j].ID
	})
	return merged
}

func (r *GormGrantRepository) Create(ctx context.Context, grant *domain.Grant) error {
	row, err := grantToRow(grant)
	if err != nil {
		return err
	}
	if row.ID == "" {
		row.ID, err = uuidx.NewV7()
		if err != nil {
			return err
		}
	}
	if row.CreatedAt == 0 {
		row.CreatedAt = timex.NowUnix()
	}
	db := r.dbForContext(ctx)
	if err := db.Create(&row).Error; err != nil {
		return err
	}
	grant.ID = row.ID
	grant.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormGrantRepository) Revoke(ctx context.Context, query domain.GrantQuery, revokedAt int64) error {
	db := r.dbForContext(ctx)
	db = applyGrantQuery(db.Model(&grantRow{}).Where("revoked_at IS NULL"), query)
	return db.Update("revoked_at", revokedAt).Error
}

func (r *GormGrantRepository) ReplaceMany(
	ctx context.Context,
	replacement domain.GrantReplacement,
	now int64,
) error {
	db := r.dbForContext(ctx)
	apply := func(tx *gorm.DB) error {
		// This revoke only targets replacement.Effect (ALLOW for every current
		// caller); an existing DENY row for the same principal, resource, and
		// action is not matched by the Effects filter and stays untouched.
		if err := applyGrantQuery(tx.Model(&grantRow{}).Where("revoked_at IS NULL"), domain.GrantQuery{
			PrincipalUserIDs: replacement.PrincipalUserIDs,
			ActionNames:      replacement.ManagedActions,
			Resource:         replacement.Resource,
			Effects:          []domain.Effect{replacement.Effect},
		}).Update("revoked_at", now).Error; err != nil {
			return err
		}
		rows, err := replacementRows(replacement, now)
		if err != nil || len(rows) == 0 {
			return err
		}
		return tx.CreateInBatches(rows, grantInsertBatchSize).Error
	}
	return db.Transaction(apply)
}

func replacementRows(replacement domain.GrantReplacement, now int64) ([]grantRow, error) {
	principalIDs := uniqueSorted(replacement.PrincipalUserIDs)
	actions := uniqueSorted(replacement.Actions)
	rows := make([]grantRow, 0, len(principalIDs)*len(actions))
	for _, principalID := range principalIDs {
		for _, action := range actions {
			grant := domain.Grant{
				PrincipalUserID: principalID, ActionName: action,
				ResourceType: replacement.Resource.Type, ResourceID: replacement.Resource.ID,
				Effect: replacement.Effect, Conditions: replacement.Conditions,
				ValidFrom: replacement.ValidFrom, ExpiresAt: replacement.ExpiresAt,
				GrantedByUserID: replacement.GrantedByUserID, CreatedAt: now,
			}
			row, err := grantToRow(&grant)
			if err != nil {
				return nil, err
			}
			row.ID, err = uuidx.NewV7()
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func applyGrantQuery(db *gorm.DB, query domain.GrantQuery) *gorm.DB {
	if len(query.PrincipalUserIDs) > 0 {
		db = db.Where("principal_user_id IN ?", query.PrincipalUserIDs)
	}
	if len(query.ActionNames) > 0 {
		db = db.Where("action_name IN ?", query.ActionNames)
	}
	if query.Resource.Type != "" {
		db = db.Where("resource_type = ?", query.Resource.Type)
	}
	if query.Resource.ID != "" {
		db = db.Where("resource_id = ?", query.Resource.ID)
	}
	if len(query.Effects) > 0 {
		effects := make([]string, len(query.Effects))
		for i, effect := range query.Effects {
			effects[i] = string(effect)
		}
		db = db.Where("effect IN ?", effects)
	}
	return db
}

func grantToRow(grant *domain.Grant) (grantRow, error) {
	conditions, err := marshalConditions(grant.Conditions)
	if err != nil {
		return grantRow{}, err
	}
	return grantRow{
		ID: grant.ID, PrincipalUserID: grant.PrincipalUserID,
		ActionName: grant.ActionName, ResourceType: grant.ResourceType, ResourceID: grant.ResourceID,
		Effect: string(grant.Effect), Conditions: conditions, ValidFrom: grant.ValidFrom,
		ExpiresAt: grant.ExpiresAt, GrantedByUserID: stringPointer(grant.GrantedByUserID),
		CreatedAt: grant.CreatedAt, RevokedAt: grant.RevokedAt,
	}, nil
}

func rowToGrant(row *grantRow) (domain.Grant, error) {
	conditions, err := unmarshalConditions(row.Conditions)
	if err != nil {
		return domain.Grant{}, err
	}
	return domain.Grant{
		ID: row.ID, PrincipalUserID: row.PrincipalUserID,
		ActionName: row.ActionName, ResourceType: row.ResourceType, ResourceID: row.ResourceID,
		Effect: domain.Effect(row.Effect), Conditions: conditions, ValidFrom: row.ValidFrom,
		ExpiresAt: row.ExpiresAt, GrantedByUserID: stringValue(row.GrantedByUserID),
		CreatedAt: row.CreatedAt, RevokedAt: row.RevokedAt,
	}, nil
}

func marshalConditions(conditions []domain.Condition) ([]byte, error) {
	payload := make([]conditionPayload, len(conditions))
	for i, condition := range conditions {
		payload[i] = conditionPayload{Operator: string(condition.Operator), Key: condition.Key, Values: condition.Values}
	}
	return json.Marshal(payload)
}

func unmarshalConditions(raw []byte) ([]domain.Condition, error) {
	if len(raw) == 0 {
		return []domain.Condition{}, nil
	}
	var payload []conditionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	conditions := make([]domain.Condition, len(payload))
	for i, condition := range payload {
		conditions[i] = domain.Condition{
			Operator: domain.ConditionOperator(condition.Operator), Key: condition.Key, Values: condition.Values,
		}
	}
	return conditions, nil
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
