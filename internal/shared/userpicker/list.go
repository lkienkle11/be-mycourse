package userpicker

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/utils"
)

// Row is a paginated user-picker list item with optional avatar fields.
type Row struct {
	UserID       string `gorm:"column:user_id"`
	DisplayName  string `gorm:"column:display_name"`
	Email        string `gorm:"column:email"`
	AvatarFileID string `gorm:"column:avatar_file_id"`
	AvatarURL    string `gorm:"column:avatar_url"`
}

// ListFilter drives pagination and search for user-picker queries.
type ListFilter struct {
	Page    int
	PerPage int
	Search  string
}

// ListRows runs count + paginated list against a caller-provided base SQL and args.
func ListRows(ctx context.Context, db *gorm.DB, baseSQL string, args map[string]any, filter ListFilter) ([]Row, int64, error) {
	parsed := utils.ParseListFilter(utils.BaseFilter{
		Page:    filter.Page,
		PerPage: filter.PerPage,
	})
	searchClause, searchArgs := utils.UserDisplayNameEmailSearchSQL(filter.Search)
	for k, v := range searchArgs {
		args[k] = v
	}
	countQ := "SELECT COUNT(*) FROM (" + baseSQL + searchClause + ") AS candidates"
	var total int64
	if err := gormRaw(db.WithContext(ctx), countQ, args).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	listQ := baseSQL + searchClause + fmt.Sprintf(" ORDER BY user_id DESC LIMIT %d OFFSET %d", parsed.PerPage, parsed.Offset)
	var rows []Row
	if err := gormRaw(db.WithContext(ctx), listQ, args).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// gormRaw avoids passing an empty named-arg map when SQL has no @placeholders (GORM/pg: "expected 0 arguments, got 1").
func gormRaw(db *gorm.DB, sql string, args map[string]any) *gorm.DB {
	if len(args) == 0 {
		return db.Raw(sql)
	}
	return db.Raw(sql, args)
}

// UserPickerSelectSQL returns the shared SELECT columns and media join for picker rows.
func UserPickerSelectSQL(usersTable string) string {
	return fmt.Sprintf(`
SELECT
    u.id AS user_id,
    COALESCE(u.display_name, '') AS display_name,
    COALESCE(u.email, '') AS email,
    COALESCE(u.avatar_file_id::text, '') AS avatar_file_id,
    COALESCE(m.url, '') AS avatar_url
FROM %s u
LEFT JOIN media_files m
    ON m.id = u.avatar_file_id AND m.deleted_at IS NULL`, usersTable)
}
