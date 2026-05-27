package infra

import (
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
)

var (
	mediaListSortColumns = map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"filename":   "filename",
		"size_bytes": "size_bytes",
	}

	documentExtensions = []string{
		".pdf", ".doc", ".docx", ".ppt", ".pptx",
		".xls", ".xlsx", ".txt", ".zip", ".rar", ".7z", ".tar", ".gz",
	}
)

// mediaListOrderClause returns a whitelisted ORDER BY clause for media file listing.
func mediaListOrderClause(sortBy, sortOrder string) string {
	col, ok := mediaListSortColumns[strings.TrimSpace(sortBy)]
	if !ok {
		col = "created_at"
	}
	order := strings.ToUpper(strings.TrimSpace(sortOrder))
	if order != "ASC" {
		order = "DESC"
	}
	return col + " " + order
}

func applyMediaCategoryFilter(q *gorm.DB, category *string) *gorm.DB {
	if category == nil {
		return q
	}
	switch strings.TrimSpace(*category) {
	case "image":
		return q.Where(imageCategorySQL)
	case "document":
		return q.Where("kind = ?", constants.FileKindFile).
			Where("NOT (" + imageCategorySQL + ")").
			Where(documentCategorySQL)
	case "video":
		return q.Where("kind = ?", constants.FileKindVideo)
	default:
		return q
	}
}

// imageCategorySQL matches IsImageMIMEOrExt (mime image/* or raster extensions).
const imageCategorySQL = `(
	LOWER(mime_type) LIKE 'image/%' OR
	LOWER(filename) LIKE '%.jpg' OR LOWER(filename) LIKE '%.jpeg' OR
	LOWER(filename) LIKE '%.png' OR LOWER(filename) LIKE '%.gif' OR
	LOWER(filename) LIKE '%.bmp' OR LOWER(filename) LIKE '%.tiff' OR
	LOWER(filename) LIKE '%.tif' OR LOWER(filename) LIKE '%.webp'
)`

// documentCategorySQL matches non-image FILE uploads classified as documents/archives.
var documentCategorySQL = buildDocumentCategorySQL()

func buildDocumentCategorySQL() string {
	parts := make([]string, 0, len(documentExtensions)+1)
	parts = append(parts, "LOWER(mime_type) = 'application/pdf'")
	for _, ext := range documentExtensions {
		parts = append(parts, "LOWER(filename) LIKE '%"+ext+"'")
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func applyMediaListFilters(q *gorm.DB, filter domain.FileFilter) *gorm.DB {
	if term, ok := mediaFilenameSearchValue(filter.Search); ok {
		q = q.Where("filename ILIKE ?", term)
	}
	if filter.Provider != nil {
		q = q.Where("provider = ?", *filter.Provider)
	}
	if filter.Kind != nil {
		q = q.Where("kind = ?", *filter.Kind)
	}
	return applyMediaCategoryFilter(q, filter.Category)
}

func mediaFilenameSearchValue(search string) (string, bool) {
	term := strings.TrimSpace(search)
	if term == "" {
		return "", false
	}
	return "%" + term + "%", true
}
