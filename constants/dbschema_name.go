package constants

// PostgreSQL relation (table) names used by GORM TableName(), dbschema accessors, and raw SQL.
// Do not import dbschema from this file — avoids import cycles (dbschema imports constants).

// --- RBAC ---

const (
	TableRBACPermissions     = "permissions"
	TableRBACRoles           = "roles"
	TableRBACRolePermissions = "role_permissions"
	TableRBACUserRoles       = "user_roles"
	TableRBACUserPermissions = "user_permissions"
)

// --- Media ---

const (
	TableMediaFiles               = "media_files"
	TableMediaPendingCloudCleanup = "media_pending_cloud_cleanup"
)

// --- Taxonomy ---

const (
	TableTaxonomyTags         = "tags"
	TableTaxonomyCategories   = "categories"
	TableTaxonomyCourseLevels = "course_levels"
)

// --- System (singleton / operators) ---

const (
	TableSystemAppConfig       = "system_app_config"
	TableSystemPrivilegedUsers = "system_privileged_users"
)

// --- Application users (custom users table, BIGINT id) ---

const TableAppUsers = "users"
