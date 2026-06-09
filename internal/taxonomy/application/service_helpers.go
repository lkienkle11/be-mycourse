package application

import (
	"context"

	"mycourse-io-be/internal/taxonomy/domain"
)

func (s *TaxonomyService) deleteWithOrphanImage(
	ctx context.Context,
	id string,
	loadImageID func(context.Context, string) (*string, error),
	deleteFn func(context.Context, string) error,
) error {
	imgID, err := loadImageID(ctx, id)
	if err != nil {
		return err
	}
	prevFileID := imageFileIDStr(imgID)
	if err := deleteFn(ctx, id); err != nil {
		return err
	}
	if prevFileID != "" && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return nil
}

type slugStatusEntityRepo[T any] struct {
	build   func(n, sl, st string, createdBy *string) *T
	idOf    func(*T) string
	create  func(context.Context, *T) error
	getByID func(context.Context, string) (*T, error)
}

func createSlugStatusEntity[T any](
	ctx context.Context,
	name, status string,
	actorID string,
	repo slugStatusEntityRepo[T],
) (*T, error) {
	n, sl, st := trimmedTaxonomyFields(name, status)
	entity := repo.build(n, sl, st, stringPtrIfNotBlank(actorID))
	if err := repo.create(ctx, entity); err != nil {
		return nil, err
	}
	return repo.getByID(ctx, repo.idOf(entity))
}

func updateSlugStatusEntity[T any, In any](
	ctx context.Context,
	id string,
	in In,
	getByID func(context.Context, string) (*T, error),
	save func(context.Context, *T) error,
	apply func(*T, In),
) (*T, error) {
	entity, err := getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	apply(entity, in)
	if err := save(ctx, entity); err != nil {
		return nil, err
	}
	return getByID(ctx, id)
}

func newTagFromFields(n, sl, st string, createdBy *string) *domain.Tag {
	return &domain.Tag{Name: n, Slug: sl, Status: st, CreatedBy: createdBy}
}

func newCourseLevelFromFields(n, sl, st string, createdBy *string) *domain.CourseLevel {
	return &domain.CourseLevel{Name: n, Slug: sl, Status: st, CreatedBy: createdBy}
}

func tagID(t *domain.Tag) string { return t.ID }

func courseLevelID(cl *domain.CourseLevel) string { return cl.ID }

type slugStatusCreator[T any] = slugStatusEntityRepo[T]

func createSlugStatusFromInput[T any](ctx context.Context, in domain.CreateTagInput, c slugStatusCreator[T]) (*T, error) {
	return createSlugStatusEntity(ctx, in.Name, in.Status, in.ActorID, c)
}

func (s *TaxonomyService) tagCreator() slugStatusCreator[domain.Tag] {
	return slugStatusCreator[domain.Tag]{newTagFromFields, tagID, s.tagRepo.Create, s.tagRepo.GetByID}
}

func (s *TaxonomyService) courseLevelCreator() slugStatusCreator[domain.CourseLevel] {
	return slugStatusCreator[domain.CourseLevel]{newCourseLevelFromFields, courseLevelID, s.courseLevelRepo.Create, s.courseLevelRepo.GetByID}
}

func updateSlugStatusRepo[T any](
	ctx context.Context,
	id string,
	in domain.UpdateTagInput,
	getByID func(context.Context, string) (*T, error),
	save func(context.Context, *T) error,
) (*T, error) {
	return updateSlugStatusEntity(ctx, id, in, getByID, save, func(entity *T, in domain.UpdateTagInput) {
		updateSlugStatusFields(entity, in)
	})
}

func updateSlugStatusFields(entity any, in domain.UpdateTagInput) {
	switch e := entity.(type) {
	case *domain.Tag:
		applyOptionalTaxonomyFields(&e.Name, &e.Slug, &e.Status, in.Name, in.Status)
	case *domain.CourseLevel:
		applyOptionalTaxonomyFields(&e.Name, &e.Slug, &e.Status, in.Name, in.Status)
	}
}

func imageIDLoader[T any](load func(context.Context, string) (*T, error), pick func(*T) *string) func(context.Context, string) (*string, error) {
	return func(ctx context.Context, id string) (*string, error) {
		row, err := load(ctx, id)
		if err != nil {
			return nil, err
		}
		return pick(row), nil
	}
}

func imageIDLoaderTopic(s *TaxonomyService) func(context.Context, string) (*string, error) {
	return imageIDLoader(s.topicRepo.GetByID, func(t *domain.CourseTopic) *string { return t.ImageFileID })
}

func imageIDLoaderOutcome(s *TaxonomyService) func(context.Context, string) (*string, error) {
	return imageIDLoader(s.outcomeRepo.GetByID, func(o *domain.CourseOutcome) *string { return o.ImageFileID })
}
