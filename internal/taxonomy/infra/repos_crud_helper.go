package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/taxonomy/domain"
)

func copyStringRowMeta(entityID *string, createdAt, updatedAt *int64, rowID string, rowCreated, rowUpdated int64) {
	*entityID = rowID
	*createdAt = rowCreated
	*updatedAt = rowUpdated
}

func rowStringTimestamps(id string, createdAt, updatedAt int64) (string, int64, int64) {
	return id, createdAt, updatedAt
}

func metaCourseTopicRow(r *courseTopicRow) (string, int64, int64) {
	return rowStringTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaCourseOutcomeRow(r *courseOutcomeRow) (string, int64, int64) {
	return rowStringTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaCourseSkillRow(r *courseSkillRow) (string, int64, int64) {
	return rowStringTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaTagRow(r *tagRow) (string, int64, int64) {
	return rowStringTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaCourseLevelRow(r *courseLevelRow) (string, int64, int64) {
	return rowStringTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func createTaxonomyDomain[Row any, Domain any](
	ctx context.Context,
	db *gorm.DB,
	d *Domain,
	toRow func(*Domain) *Row,
	entityID *string,
	createdAt, updatedAt *int64,
	meta func(*Row) (string, int64, int64),
) error {
	return taxonomyCreateSync(ctx, db, toRow(d), entityID, createdAt, updatedAt, meta)
}

func taxonomyCreateSync[Row any](
	ctx context.Context,
	db *gorm.DB,
	row *Row,
	entityID *string,
	createdAt, updatedAt *int64,
	meta func(*Row) (string, int64, int64),
) error {
	if err := ensureRowPrimaryKey(row); err != nil {
		return err
	}
	gormx.TouchCreatedUpdated(createdAt, updatedAt)
	return taxonomyCreate(ctx, db, row, func(r *Row) {
		id, c, u := meta(r)
		copyStringRowMeta(entityID, createdAt, updatedAt, id, c, u)
	})
}

func ensureRowPrimaryKey(row any) error {
	switch r := row.(type) {
	case *courseTopicRow:
		return gormx.EnsureStringID(&r.ID)
	case *courseOutcomeRow:
		return gormx.EnsureStringID(&r.ID)
	case *courseSkillRow:
		return gormx.EnsureStringID(&r.ID)
	case *tagRow:
		return gormx.EnsureStringID(&r.ID)
	case *courseLevelRow:
		return gormx.EnsureStringID(&r.ID)
	default:
		return nil
	}
}

func taxonomyList[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	model *R,
	filter domain.TaxonomyFilter,
	applySearch func(*gorm.DB, domain.TaxonomyFilter) *gorm.DB,
	mapRow func(*R) D,
) ([]D, int64, error) {
	q := db.WithContext(ctx).Model(model)
	if filter.IncludeDeleted {
		q = gormx.ScopeIncludeDeleted(q)
	} else {
		q = gormx.ScopeActiveOnly(q)
	}
	q = applySearch(q, filter)
	var rows []R
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	total, err := taxonomyListTotal(q, filter, len(rows))
	if err != nil {
		return nil, 0, err
	}
	out := make([]D, len(rows))
	for i := range rows {
		out[i] = mapRow(&rows[i])
	}
	return out, total, nil
}

// taxonomyListTotal avoids a separate COUNT query when the page is clearly the last one.
func taxonomyListTotal(q *gorm.DB, filter domain.TaxonomyFilter, rowCount int) (int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if rowCount < pageSize {
		return int64((page-1)*pageSize + rowCount), nil
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func taxonomyGetByID[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	id string,
	mapRow func(*R) D,
) (*D, error) {
	var row R
	q := gormx.ScopeActiveOnly(db.WithContext(ctx))
	if err := q.First(&row, "id = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	out := mapRow(&row)
	return &out, nil
}

func taxonomyCreate[R any](
	ctx context.Context,
	db *gorm.DB,
	row *R,
	sync func(*R),
) error {
	return gormx.CreateAndThen(ctx, db, row, func() { sync(row) })
}
