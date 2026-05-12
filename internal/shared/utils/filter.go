// Package utils — filter.go contains BaseFilter and list-filter parsing utilities
// migrated from dto/filter.go and pkg/query/filter_parser.go.
package utils

import "strings"

// BaseFilter is the mandatory embed for every GET-list query-parameter DTO.
// All 6 fields are optional — callers may omit any of them and sensible defaults
// will be applied by the helper methods.
type BaseFilter struct {
	Page       int    `form:"page"`
	PerPage    int    `form:"per_page"`
	SortBy     string `form:"sort_by"`
	SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
	SearchBy   string `form:"search_by"`
	SearchData string `form:"search_data"`
}

func (f BaseFilter) GetPage() int {
	if f.Page < 1 {
		return 1
	}
	return f.Page
}

func (f BaseFilter) GetPerPage() int {
	if f.PerPage < 1 {
		return 20
	}
	if f.PerPage > 100 {
		return 100
	}
	return f.PerPage
}

func (f BaseFilter) GetOffset() int {
	return (f.GetPage() - 1) * f.GetPerPage()
}

func (f BaseFilter) GetSortOrder() string {
	if f.SortOrder == "desc" {
		return "desc"
	}
	return "asc"
}

func (f BaseFilter) HasSearch() bool {
	return f.SearchBy != "" && f.SearchData != ""
}

func (f BaseFilter) HasSort() bool {
	return f.SortBy != ""
}

// ParsedListFilter carries resolved pagination values from a BaseFilter.
type ParsedListFilter struct {
	Page    int
	PerPage int
	Offset  int
}

// ParseListFilter resolves pagination defaults from a BaseFilter.
func ParseListFilter(base BaseFilter) ParsedListFilter {
	return ParsedListFilter{
		Page:    base.GetPage(),
		PerPage: base.GetPerPage(),
		Offset:  base.GetOffset(),
	}
}

// BuildSortClause produces a safe "column ASC|DESC" fragment.
// allowed maps query field names → real column names; fallback is used when SortBy is unrecognised.
func BuildSortClause(base BaseFilter, allowed map[string]string, fallback string) string {
	col := fallback
	if c, ok := allowed[base.SortBy]; ok {
		col = c
	}
	order := "ASC"
	if base.GetSortOrder() == "desc" {
		order = "DESC"
	}
	return col + " " + order
}

// BuildSearchClause returns the ILIKE predicate column, value, and true when search is applicable.
func BuildSearchClause(base BaseFilter, allowed map[string]string) (string, string, bool) {
	if !base.HasSearch() {
		return "", "", false
	}
	col, ok := allowed[base.SearchBy]
	if !ok {
		return "", "", false
	}
	term := strings.TrimSpace(base.SearchData)
	if term == "" {
		return "", "", false
	}
	return col + " ILIKE ?", "%" + term + "%", true
}
