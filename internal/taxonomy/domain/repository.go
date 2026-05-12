package domain

import "context"

// CategoryRepository defines persistence for the Category aggregate.
type CategoryRepository interface {
	List(ctx context.Context, filter TaxonomyFilter) ([]Category, int64, error)
	GetByID(ctx context.Context, id uint) (*Category, error)
	Create(ctx context.Context, c *Category) error
	Save(ctx context.Context, c *Category) error
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
