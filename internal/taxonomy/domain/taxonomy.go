// Package domain contains the TAXONOMY bounded-context core entities and repository interfaces.
package domain

import (
	taxpkg "mycourse-io-be/pkg/taxonomy"
)

// CourseTopic is the aggregate root for a taxonomy course topic (formerly category).
type CourseTopic struct {
	ID          uint
	Name        string
	Slug        string
	ImageFileID *string
	ChildTopics []taxpkg.TreeNode
	Status      string
	CreatedBy   *uint
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   *int64

	ImageFileURL  string
	ImageFileKind string
	ImageFileMime string
}

// CourseOutcome is the aggregate root for a course learning outcome.
type CourseOutcome struct {
	ID               uint
	ShortDescription string
	Description      []string
	ImageFileID      *string
	Status           string
	CreatedBy        *uint
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64

	ImageFileURL  string
	ImageFileKind string
	ImageFileMime string
}

// CourseSkill is the aggregate root for a course skill tree root row.
type CourseSkill struct {
	ID        uint
	Name      string
	Slug      string
	Children  []taxpkg.TreeNode
	Status    string
	CreatedBy *uint
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

// Tag is the aggregate root for a taxonomy tag.
type Tag struct {
	ID        uint
	Name      string
	Slug      string
	Status    string
	CreatedBy *uint
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

// CourseLevel is the aggregate root for a taxonomy course level.
type CourseLevel struct {
	ID        uint
	Name      string
	Slug      string
	Status    string
	CreatedBy *uint
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

// TaxonomyFilter is the common filter for all taxonomy list queries.
type TaxonomyFilter struct {
	Page           int
	PageSize       int
	Status         *string
	SearchBy       string
	SearchValue    string
	SortBy         string
	SortDesc       bool
	IncludeDeleted bool // true for GET .../full list routes
}

// CreateCourseTopicInput carries data for creating a new course topic.
type CreateCourseTopicInput struct {
	ActorID     uint
	Name        string
	Slug        string
	Status      string
	ImageFileID string
	ChildTopics []taxpkg.TreeNode
}

// UpdateCourseTopicInput carries partial-update data for a course topic.
type UpdateCourseTopicInput struct {
	Name        *string
	Slug        *string
	Status      *string
	ImageFileID *string
	ChildTopics *[]taxpkg.TreeNode
}

// CreateCourseOutcomeInput carries data for creating a course outcome.
type CreateCourseOutcomeInput struct {
	ActorID          uint
	ShortDescription string
	Description      []string
	Status           string
	ImageFileID      string
}

// UpdateCourseOutcomeInput carries partial-update data for a course outcome.
type UpdateCourseOutcomeInput struct {
	ShortDescription *string
	Description      *[]string
	Status           *string
	ImageFileID      *string
}

// CreateCourseSkillInput carries data for creating a course skill.
type CreateCourseSkillInput struct {
	ActorID  uint
	Name     string
	Slug     string
	Status   string
	Children []taxpkg.TreeNode
}

// UpdateCourseSkillInput carries partial-update data for a course skill.
type UpdateCourseSkillInput struct {
	Name     *string
	Slug     *string
	Status   *string
	Children *[]taxpkg.TreeNode
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
type CreateCourseLevelInput = CreateTagInput

// UpdateCourseLevelInput carries partial-update data for a course level.
type UpdateCourseLevelInput = UpdateTagInput
