package domain

import "context"

// taxonomyRepository is the shared CRUD contract for taxonomy aggregates.
type taxonomyRepository[T any] interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]T, int64, error)
	GetByID(ctx context.Context, id uint) (*T, error)
	Create(ctx context.Context, t *T) error
	Save(ctx context.Context, t *T) error
	Delete(ctx context.Context, id uint) error
}

// CourseTopicRepository defines persistence for the CourseTopic aggregate.
type CourseTopicRepository taxonomyRepository[CourseTopic]

// CourseOutcomeRepository defines persistence for the CourseOutcome aggregate.
type CourseOutcomeRepository taxonomyRepository[CourseOutcome]

// CourseSkillRepository defines persistence for the CourseSkill aggregate.
type CourseSkillRepository taxonomyRepository[CourseSkill]

// TagRepository defines persistence for the Tag aggregate.
type TagRepository taxonomyRepository[Tag]

// CourseLevelRepository defines persistence for the CourseLevel aggregate.
type CourseLevelRepository taxonomyRepository[CourseLevel]
