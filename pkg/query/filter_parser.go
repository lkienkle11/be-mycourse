package query

import (
	"strings"

	"mycourse-io-be/dto"
)

type ParsedListFilter struct {
	Page    int
	PerPage int
	Offset  int
}

func ParseListFilter(base dto.BaseFilter) ParsedListFilter {
	return ParsedListFilter{
		Page:    base.GetPage(),
		PerPage: base.GetPerPage(),
		Offset:  base.GetOffset(),
	}
}

func BuildSortClause(base dto.BaseFilter, allowed map[string]string, fallback string) string {
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

func BuildSearchClause(base dto.BaseFilter, allowed map[string]string) (string, string, bool) {
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
