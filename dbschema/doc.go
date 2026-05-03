// Package dbschema exposes PostgreSQL table (relation) names per business module.
//
// Authoritative string literals live in [constants/dbschema_name.go] (single source of truth).
// This package only provides typed namespaces (e.g. RBAC.Permissions(), Media.Files()) so
// models and services avoid scattering table-name strings.
//
// GORM struct tags that require literals (many2many) must use the same spelling as the
// corresponding RBAC.*() value — see comments on models.
package dbschema
