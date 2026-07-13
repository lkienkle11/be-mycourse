package infra

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

// --- row mappers -------------------------------------------------------------

type imageFileRow struct {
	ID   string `gorm:"column:id"`
	URL  string `gorm:"column:url"`
	Kind string `gorm:"column:kind"`
	Mime string `gorm:"column:mime_type"`
}

func topicImageID(row *domain.CourseTopic) *string { return row.ImageFileID }

func outcomeImageID(row *domain.CourseOutcome) *string { return row.ImageFileID }

func applyTopicImageRow(row *domain.CourseTopic, img imageFileRow) {
	row.ImageFileURL = img.URL
	row.ImageFileKind = img.Kind
	row.ImageFileMime = img.Mime
}

func applyOutcomeImageRow(row *domain.CourseOutcome, img imageFileRow) {
	row.ImageFileURL = img.URL
	row.ImageFileKind = img.Kind
	row.ImageFileMime = img.Mime
}

func trimStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func collectImageFileIDs[T any](rows []T, getID func(*T) *string) []string {
	ids := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for i := range rows {
		id := trimStringPtr(getID(&rows[i]))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func loadImageFileMap(ctx context.Context, db *gorm.DB, ids []string) (map[string]imageFileRow, error) {
	rows, err := gormx.FindActiveByIDs[imageFileRow](ctx, db, constants.TableMediaFiles,
		"id, url, kind, mime_type", ids)
	if err != nil {
		return nil, err
	}
	return gormx.IndexByID(rows, func(row imageFileRow) string { return row.ID }), nil
}

func hydrateImageURLs[T any](
	ctx context.Context,
	db *gorm.DB,
	rows []T,
	getID func(*T) *string,
	setImage func(*T, imageFileRow),
) ([]T, error) {
	ids := collectImageFileIDs(rows, getID)
	images, err := loadImageFileMap(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		id := trimStringPtr(getID(&rows[i]))
		img, ok := images[id]
		if !ok {
			continue
		}
		setImage(&rows[i], img)
	}
	return rows, nil
}

func listTaxonomyWithImageURLs[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	model *R,
	filter domain.TaxonomyFilter,
	applySearch func(*gorm.DB, domain.TaxonomyFilter) *gorm.DB,
	mapRow func(*R) D,
	getID func(*D) *string,
	setImage func(*D, imageFileRow),
) ([]D, int64, error) {
	rows, total, err := taxonomyList(ctx, db, model, filter, applySearch, mapRow)
	if err != nil {
		return nil, 0, err
	}
	if !filter.IncludeImages {
		return rows, total, nil
	}
	rows, err = hydrateImageURLs(ctx, db, rows, getID, setImage)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func taxonomyGetByIDWithImageURLs[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	id string,
	mapRow func(*R) D,
	getID func(*D) *string,
	setImage func(*D, imageFileRow),
) (*D, error) {
	row, err := taxonomyGetByID(ctx, db, id, mapRow)
	if err != nil {
		return nil, err
	}
	rows, err := hydrateImageURLs(ctx, db, []D{*row}, getID, setImage)
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

type listHydratedWithImagesArgs[R any, D any] struct {
	Model       *R
	Filter      domain.TaxonomyFilter
	ApplySearch func(*gorm.DB, domain.TaxonomyFilter) *gorm.DB
	MapRow      func(*R) D
	GetID       func(*D) *string
	SetImage    func(*D, imageFileRow)
	Hydrate     func(context.Context, *gorm.DB, string, []D) error
}

func listHydratedWithImages[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	args listHydratedWithImagesArgs[R, D],
) ([]D, int64, error) {
	items, total, err := listTaxonomyWithImageURLs(ctx, db, args.Model, args.Filter, args.ApplySearch, args.MapRow, args.GetID, args.SetImage)
	if err != nil {
		return nil, 0, err
	}
	if err := args.Hydrate(ctx, db, args.Filter.Locale, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func listHydrated[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	model *R,
	filter domain.TaxonomyFilter,
	applySearch func(*gorm.DB, domain.TaxonomyFilter) *gorm.DB,
	mapRow func(*R) D,
	hydrate func(context.Context, *gorm.DB, string, []D) error,
) ([]D, int64, error) {
	items, total, err := taxonomyList(ctx, db, model, filter, applySearch, mapRow)
	if err != nil {
		return nil, 0, err
	}
	if err := hydrate(ctx, db, filter.Locale, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type createNameTranslationArgs[Row any, Domain any] struct {
	Entity       *Domain
	RowVersion   *int64
	ToRow        func(*Domain) *Row
	EntityID     *string
	CreatedAt    *int64
	UpdatedAt    *int64
	Meta         func(*Row) (string, int64, int64)
	Table        string
	FKCol        string
	Translations map[string]taxpkg.NodeTranslation
}

func createWithNameTranslations[Row any, Domain any](
	ctx context.Context,
	db *gorm.DB,
	args createNameTranslationArgs[Row, Domain],
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if *args.RowVersion == 0 {
			*args.RowVersion = 1
		}
		if err := createTaxonomyDomain(ctx, tx, args.Entity, args.ToRow, args.EntityID, args.CreatedAt, args.UpdatedAt, args.Meta); err != nil {
			return err
		}
		return upsertNameTranslations(ctx, tx, args.Table, args.FKCol, *args.EntityID, args.Translations)
	})
}

func rowToCourseTopic(r *courseTopicRow, img *imageFileRow) domain.CourseTopic {
	child := []taxpkg.TreeNode(r.ChildTopics)
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	t := domain.CourseTopic{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		ImageFileID: r.ImageFileID, ChildTopics: child, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
		RowVersion: r.RowVersion,
	}
	if img != nil {
		t.ImageFileURL = img.URL
		t.ImageFileKind = img.Kind
		t.ImageFileMime = img.Mime
	}
	return t
}

func courseTopicToRow(t *domain.CourseTopic) *courseTopicRow {
	child := t.ChildTopics
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return &courseTopicRow{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		ImageFileID: t.ImageFileID, ChildTopics: treeNodesJSONB(child),
		RowVersion: t.RowVersion,
		CreatedBy:  t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, DeletedAt: t.DeletedAt,
	}
}

func rowToCourseOutcome(r *courseOutcomeRow, img *imageFileRow) domain.CourseOutcome {
	desc := []string(r.Description)
	if desc == nil {
		desc = []string{}
	}
	o := domain.CourseOutcome{
		ID: r.ID, ShortDescription: r.ShortDescription, Description: desc,
		Status: r.Status, ImageFileID: r.ImageFileID, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
		RowVersion: r.RowVersion,
	}
	if img != nil {
		o.ImageFileURL = img.URL
		o.ImageFileKind = img.Kind
		o.ImageFileMime = img.Mime
	}
	return o
}

func courseOutcomeToRow(o *domain.CourseOutcome) *courseOutcomeRow {
	desc := o.Description
	if desc == nil {
		desc = []string{}
	}
	return &courseOutcomeRow{
		ID: o.ID, ShortDescription: o.ShortDescription, Description: descriptionJSONB(desc),
		Status: o.Status, ImageFileID: o.ImageFileID, CreatedBy: o.CreatedBy,
		RowVersion: o.RowVersion,
		CreatedAt:  o.CreatedAt, UpdatedAt: o.UpdatedAt, DeletedAt: o.DeletedAt,
	}
}

func rowToCourseSkill(r *courseSkillRow) domain.CourseSkill {
	child := []taxpkg.TreeNode(r.Children)
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return domain.CourseSkill{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Children: child, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
		RowVersion: r.RowVersion,
	}
}

func courseSkillToRow(s *domain.CourseSkill) *courseSkillRow {
	child := s.Children
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return &courseSkillRow{
		ID: s.ID, Name: s.Name, Slug: s.Slug, Children: treeNodesJSONB(child),
		Status: s.Status, RowVersion: s.RowVersion, CreatedBy: s.CreatedBy,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt, DeletedAt: s.DeletedAt,
	}
}

func rowToTag(r *tagRow) domain.Tag {
	return domain.Tag{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status, RowVersion: r.RowVersion,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
	}
}

func tagToRow(t *domain.Tag) *tagRow {
	return &tagRow{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status, RowVersion: t.RowVersion,
		CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, DeletedAt: t.DeletedAt,
	}
}

func rowToCourseLevel(r *courseLevelRow) domain.CourseLevel {
	cl := domain.CourseLevel{ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status}
	cl.RowVersion = r.RowVersion
	cl.CreatedBy = r.CreatedBy
	cl.CreatedAt = r.CreatedAt
	cl.UpdatedAt = r.UpdatedAt
	cl.DeletedAt = r.DeletedAt
	return cl
}

func courseLevelToRow(cl *domain.CourseLevel) *courseLevelRow {
	row := &courseLevelRow{ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status}
	row.RowVersion = cl.RowVersion
	row.CreatedBy = cl.CreatedBy
	row.CreatedAt = cl.CreatedAt
	row.UpdatedAt = cl.UpdatedAt
	row.DeletedAt = cl.DeletedAt
	return row
}
