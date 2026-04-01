package dto

// BaseFilter is the mandatory embed for every GET-list query-parameter DTO.
//
// All 6 fields are optional — callers may omit any of them and sensible defaults
// will be applied by the helper methods.
//
// Any filter struct that will be used as query params for a list API MUST embed
// BaseFilter. This guarantees that every list endpoint supports the same baseline
// pagination, sorting, and search capabilities.
//
// Usage — define a custom filter:
//
//	type UserFilter struct {
//	    dto.BaseFilter               // required embed
//	    Role   string `form:"role"`  // extra fields go here
//	    Status string `form:"status"`
//	}
//
// Bind it in a handler:
//
//	var q dto.UserFilter
//	if err := c.ShouldBindQuery(&q); err != nil {
//	    response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
//	    return
//	}
//	offset := q.GetOffset()     // (page-1) * per_page
//	limit  := q.GetPerPage()
//	order  := q.GetSortOrder()  // "asc" | "desc"
type BaseFilter struct {
	// Page is the 1-based page number. Defaults to 1 when ≤ 0.
	Page int `form:"page"`

	// PerPage is the number of items per page. Defaults to 20; capped at 100.
	PerPage int `form:"per_page"`

	// SortBy is the field name to sort by (e.g. "created_at", "name").
	// The handler is responsible for whitelisting safe column names before
	// passing this value to the database layer.
	SortBy string `form:"sort_by"`

	// SortOrder controls ascending / descending sort.
	// Accepted values: "asc", "desc". Any other value falls back to "asc".
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`

	// SearchBy is the field name to run the full-text / LIKE search against.
	// The handler decides which columns are searchable.
	SearchBy string `form:"search_by"`

	// SearchData is the search term. Used together with SearchBy.
	SearchData string `form:"search_data"`
}

// GetPage returns the current page number, defaulting to 1 when not set or invalid.
func (f *BaseFilter) GetPage() int {
	if f.Page < 1 {
		return 1
	}
	return f.Page
}

// GetPerPage returns the page size, defaulting to 20 and capping at 100.
func (f *BaseFilter) GetPerPage() int {
	if f.PerPage < 1 {
		return 20
	}
	if f.PerPage > 100 {
		return 100
	}
	return f.PerPage
}

// GetOffset returns the SQL OFFSET value derived from page and per_page.
// Use alongside GetPerPage() as LIMIT.
func (f *BaseFilter) GetOffset() int {
	return (f.GetPage() - 1) * f.GetPerPage()
}

// GetSortOrder returns a safe sort direction string ("asc" or "desc").
// Defaults to "asc" when SortOrder is empty or contains an invalid value.
func (f *BaseFilter) GetSortOrder() string {
	if f.SortOrder == "desc" {
		return "desc"
	}
	return "asc"
}

// HasSearch reports whether both SearchBy and SearchData are non-empty,
// meaning the handler should apply a search predicate.
func (f *BaseFilter) HasSearch() bool {
	return f.SearchBy != "" && f.SearchData != ""
}

// HasSort reports whether SortBy is non-empty, meaning the handler should
// apply an ORDER BY clause.
func (f *BaseFilter) HasSort() bool {
	return f.SortBy != ""
}
