package domain

import "context"

// CourseTopicRepository defines persistence for the CourseTopic aggregate.
type CourseTopicRepository interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]CourseTopic, int64, error)
	GetByID(ctx context.Context, id uint) (*CourseTopic, error)
	Create(ctx context.Context, t *CourseTopic) error
	Save(ctx context.Context, t *CourseTopic) error
	Delete(ctx context.Context, id uint) error
}

// CourseOutcomeRepository defines persistence for the CourseOutcome aggregate.
type CourseOutcomeRepository interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]CourseOutcome, int64, error)
	GetByID(ctx context.Context, id uint) (*CourseOutcome, error)
	Create(ctx context.Context, o *CourseOutcome) error
	Save(ctx context.Context, o *CourseOutcome) error
	Delete(ctx context.Context, id uint) error
}

// CourseSkillRepository defines persistence for the CourseSkill aggregate.
type CourseSkillRepository interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]CourseSkill, int64, error)
	GetByID(ctx context.Context, id uint) (*CourseSkill, error)
	Create(ctx context.Context, s *CourseSkill) error
	Save(ctx context.Context, s *CourseSkill) error
	Delete(ctx context.Context, id uint) error
}

// TagRepository defines persistence for the Tag aggregate.
type TagRepository interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]Tag, int64, error)
	GetByID(ctx context.Context, id uint) (*Tag, error)
	Create(ctx context.Context, t *Tag) error
	Save(ctx context.Context, t *Tag) error
	Delete(ctx context.Context, id uint) error
}

// CourseLevelRepository defines persistence for the CourseLevel aggregate.
type CourseLevelRepository interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]CourseLevel, int64, error)
	GetByID(ctx context.Context, id uint) (*CourseLevel, error)
	Create(ctx context.Context, cl *CourseLevel) error
	Save(ctx context.Context, cl *CourseLevel) error
	Delete(ctx context.Context, id uint) error
}
