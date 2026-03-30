// Package dbschema centralizes PostgreSQL table (relation) names per business module.
//
// Each domain lives in its own file (e.g. rbac.go) with:
//   - unexported string constants (single source of truth)
//   - a namespace value (e.g. RBAC) with methods like RBAC.Permissions() for call sites
//
// GORM struct tags that require literals (many2many) must use the same spelling as the
// corresponding RBAC.*() value — see comments on models.
package dbschema
