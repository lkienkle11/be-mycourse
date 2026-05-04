package entities

// ListPermissionsParams carries validated filter values for RBAC permission listing (from the HTTP layer).
type ListPermissionsParams struct {
	Offset     int
	Limit      int
	SortBy     string
	SortOrder  string // "asc" | "desc"
	SearchBy   string
	SearchData string
}
