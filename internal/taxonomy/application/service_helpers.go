package application

import (
	"context"
	"strings"

	taxpkg "mycourse-io-be/internal/shared/taxonomy"
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
	build     func(n, sl, st string, createdBy *string, tr map[string]taxpkg.NodeTranslation) *T
	idOf      func(*T) string
	create    func(context.Context, *T) error
	getDetail func(context.Context, string, string, bool) (*T, error)
}

func createSlugStatusEntity[T any](
	ctx context.Context,
	name, status string,
	actorID string,
	translations map[string]taxpkg.NodeTranslation,
	repo slugStatusEntityRepo[T],
) (*T, error) {
	n, tr, err := syncNameCanonicalAndTranslations(name, translations)
	if err != nil {
		return nil, err
	}
	_, sl, st := trimmedTaxonomyFields(n, status)
	entity := repo.build(n, sl, st, stringPtrIfNotBlank(actorID), tr)
	if err := repo.create(ctx, entity); err != nil {
		return nil, err
	}
	return repo.getDetail(ctx, repo.idOf(entity), "", true)
}

func newTagFromFields(n, sl, st string, createdBy *string, tr map[string]taxpkg.NodeTranslation) *domain.Tag {
	return &domain.Tag{Name: n, Slug: sl, Status: st, CreatedBy: createdBy, Translations: tr, RowVersion: 1}
}

func newCourseLevelFromFields(n, sl, st string, createdBy *string, tr map[string]taxpkg.NodeTranslation) *domain.CourseLevel {
	return &domain.CourseLevel{Name: n, Slug: sl, Status: st, CreatedBy: createdBy, Translations: tr, RowVersion: 1}
}

func tagID(t *domain.Tag) string { return t.ID }

func courseLevelID(cl *domain.CourseLevel) string { return cl.ID }

type slugStatusCreator[T any] = slugStatusEntityRepo[T]

func createSlugStatusFromInput[T any](ctx context.Context, in domain.CreateTagInput, c slugStatusCreator[T]) (*T, error) {
	return createSlugStatusEntity(ctx, in.Name, in.Status, in.ActorID, in.Translations, c)
}

func (s *TaxonomyService) tagCreator() slugStatusCreator[domain.Tag] {
	return slugStatusCreator[domain.Tag]{newTagFromFields, tagID, s.tagRepo.Create, s.tagRepo.GetDetail}
}

func (s *TaxonomyService) courseLevelCreator() slugStatusCreator[domain.CourseLevel] {
	return slugStatusCreator[domain.CourseLevel]{newCourseLevelFromFields, courseLevelID, s.courseLevelRepo.Create, s.courseLevelRepo.GetDetail}
}

func updateSlugStatusRepo[T any](
	ctx context.Context,
	id string,
	in domain.UpdateTagInput,
	getDetail func(context.Context, string, string, bool) (*T, error),
	save func(context.Context, *T, int64) error,
	apply func(*T, domain.UpdateTagInput) error,
) (*T, error) {
	entity, err := getDetail(ctx, id, "", true)
	if err != nil {
		return nil, err
	}
	if err := apply(entity, in); err != nil {
		return nil, err
	}
	if err := save(ctx, entity, in.ExpectedRowVersion); err != nil {
		return nil, err
	}
	return getDetail(ctx, id, "", true)
}

func applySlugStatusFields(
	name, slug, status *string,
	translations *map[string]taxpkg.NodeTranslation,
	in domain.UpdateTagInput,
) error {
	canonical := *name
	if in.Name != nil {
		canonical = *in.Name
	}
	merged := *translations
	if in.Translations != nil {
		patch, err := canonicalizeNameTranslations(in.Translations)
		if err != nil {
			return err
		}
		merged = replaceNameTranslations(patch)
	}
	syncedName, tr, err := syncNameCanonicalAndTranslations(canonical, merged)
	if err != nil {
		return err
	}
	nextStatus := *status
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			nextStatus = v
		}
	}
	*name = syncedName
	*slug = slugFromName(syncedName)
	*status = nextStatus
	*translations = tr
	return nil
}

func applyTagUpdate(e *domain.Tag, in domain.UpdateTagInput) error {
	return applySlugStatusFields(&e.Name, &e.Slug, &e.Status, &e.Translations, in)
}

func applyCourseLevelUpdate(e *domain.CourseLevel, in domain.UpdateTagInput) error {
	return applySlugStatusFields(&e.Name, &e.Slug, &e.Status, &e.Translations, in)
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
