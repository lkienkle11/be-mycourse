package utils

import "mycourse-io-be/internal/shared/response"

// BuildPage creates a normalized pagination response block.
// totalPages is always at least 1 to keep response shape stable.
func BuildPage(page, perPage int, totalItems int64) response.PageInfo {
	totalPages := int((totalItems + int64(perPage) - 1) / int64(perPage))
	if totalPages == 0 {
		totalPages = 1
	}

	return response.PageInfo{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
	}
}
