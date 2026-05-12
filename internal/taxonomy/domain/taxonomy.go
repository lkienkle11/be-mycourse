// Package domain contains the TAXONOMY bounded-context core entities and repository interfaces.
package domain

import "time"

// Category is the aggregate root for a taxonomy category.
type Category struct {
	ID          uint
	Name        string
	Slug        string
	ImageFileID *string
	Status      string
	CreatedBy   *uint
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// ImageFile is eagerly loaded from the media bounded context when present.
	ImageFileURL  string
	ImageFileKind string
	ImageFileMime string
}

// Tag is the aggregate root for a taxonomy tag.
type Tag struct {
	ID        uint
	Name      string
	Slug      string
	Status    string
	CreatedBy *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CourseLevel is the aggregate root for a taxonomy course level.
type CourseLevel struct {
	ID        uint
	Name      string
	Slug      string
	Status    string
	CreatedBy *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TaxonomyFilter is the common filter for all taxonomy list queries.
type TaxonomyFilter struct {
	Page     int
	PageSize int
	Status   *string
	Search   string
	SortBy   string
	SortDesc bool
}

// CreateCategoryInput carries data for creating a new category.
type CreateCategoryInput struct {
	ActorID     uint
	Name        string
	Slug        string
	Status      string
	ImageFileID string
}

// UpdateCategoryInput carries partial-update data for a category.
type UpdateCategoryInput struct {
	Name        *string
	Slug        *string
	Status      *string
	ImageFileID *string
}

// CreateTagInput carries data for creating a new tag.
type CreateTagInput struct {
	ActorID uint
	Name    string
	Slug    string
	Status  string
}

// UpdateTagInput carries partial-update data for a tag.
type UpdateTagInput struct {
	Name   *string
	Slug   *string
	Status *string
}

// CreateCourseLevelInput carries data for creating a new course level.
type CreateCourseLevelInput struct {
	ActorID uint
	Name    string
	Slug    string
	Status  string
}

// UpdateCourseLevelInput carries partial-update data for a course level.
type UpdateCourseLevelInput struct {
	Name   *string
	Slug   *string
	Status *string
}
