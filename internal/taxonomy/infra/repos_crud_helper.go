package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/taxonomy/domain"
)

func copyUintRowMeta(entityID *uint, createdAt, updatedAt *int64, rowID uint, rowCreated, rowUpdated int64) {
	*entityID = rowID
	*createdAt = rowCreated
	*updatedAt = rowUpdated
}

func rowUintTimestamps(id uint, createdAt, updatedAt int64) (uint, int64, int64) {
	return id, createdAt, updatedAt
}

func metaCourseTopicRow(r *courseTopicRow) (uint, int64, int64) {
	return rowUintTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaCourseOutcomeRow(r *courseOutcomeRow) (uint, int64, int64) {
	return rowUintTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaCourseSkillRow(r *courseSkillRow) (uint, int64, int64) {
	return rowUintTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaTagRow(r *tagRow) (uint, int64, int64) {
	return rowUintTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func metaCourseLevelRow(r *courseLevelRow) (uint, int64, int64) {
	return rowUintTimestamps(r.ID, r.CreatedAt, r.UpdatedAt)
}

func createTaxonomyDomain[Row any, Domain any](
	ctx context.Context,
	db *gorm.DB,
	d *Domain,
	toRow func(*Domain) *Row,
	entityID *uint,
	createdAt, updatedAt *int64,
	meta func(*Row) (uint, int64, int64),
) error {
	return taxonomyCreateSync(ctx, db, toRow(d), entityID, createdAt, updatedAt, meta)
}

func taxonomyCreateSync[Row any](
	ctx context.Context,
	db *gorm.DB,
	row *Row,
	entityID *uint,
	createdAt, updatedAt *int64,
	meta func(*Row) (uint, int64, int64),
) error {
	gormx.TouchCreatedUpdated(createdAt, updatedAt)
	return taxonomyCreate(ctx, db, row, func(r *Row) {
		id, c, u := meta(r)
		copyUintRowMeta(entityID, createdAt, updatedAt, id, c, u)
	})
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
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []R
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]D, len(rows))
	for i := range rows {
		out[i] = mapRow(&rows[i])
	}
	return out, total, nil
}

func taxonomyGetByID[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	id uint,
	mapRow func(*R) D,
) (*D, error) {
	var row R
	q := gormx.ScopeActiveOnly(db.WithContext(ctx))
	if err := q.First(&row, id).Error; err != nil {
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
