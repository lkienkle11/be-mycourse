// Package domain contains the TAXONOMY bounded-context core entities and repository interfaces.
package domain

import (
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
)

// OutcomeTranslation is the per-locale payload for course outcome translations.
type OutcomeTranslation struct {
	ShortDescription string   `json:"short_description"`
	Description      []string `json:"description"`
}

// CourseTopic is the aggregate root for a taxonomy course topic (formerly category).
type CourseTopic struct {
	ID          string
	Name        string
	Slug        string
	ImageFileID *string
	ChildTopics []taxpkg.TreeNode
	Status      string
	CreatedBy   *string
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   *int64

	ImageFileURL     string
	ImageFileKind    string
	ImageFileMime    string
	ResolvedLocale   string
	RowVersion       int64
	Translations     map[string]taxpkg.NodeTranslation
	AvailableLocales []string
}

// CourseOutcome is the aggregate root for a course learning outcome.
type CourseOutcome struct {
	ID               string
	ShortDescription string
	Description      []string
	ImageFileID      *string
	Status           string
	CreatedBy        *string
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64

	ImageFileURL     string
	ImageFileKind    string
	ImageFileMime    string
	ResolvedLocale   string
	RowVersion       int64
	Translations     map[string]OutcomeTranslation
	AvailableLocales []string
}

// CourseSkill is the aggregate root for a course skill tree root row.
type CourseSkill struct {
	ID               string
	Name             string
	Slug             string
	Children         []taxpkg.TreeNode
	Status           string
	CreatedBy        *string
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64
	ResolvedLocale   string
	RowVersion       int64
	Translations     map[string]taxpkg.NodeTranslation
	AvailableLocales []string
}

// Tag is the aggregate root for a taxonomy tag.
type Tag struct {
	ID               string
	Name             string
	Slug             string
	Status           string
	CreatedBy        *string
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64
	ResolvedLocale   string
	RowVersion       int64
	Translations     map[string]taxpkg.NodeTranslation
	AvailableLocales []string
}

// CourseLevel is the aggregate root for a taxonomy course level.
type CourseLevel struct {
	ID               string
	Name             string
	Slug             string
	Status           string
	CreatedBy        *string
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64
	ResolvedLocale   string
	RowVersion       int64
	Translations     map[string]taxpkg.NodeTranslation
	AvailableLocales []string
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
	IncludeImages  bool // false skips media_files join for list pickers
	Locale         string
}

// CreateCourseTopicInput carries data for creating a new course topic.
type CreateCourseTopicInput struct {
	ActorID      string
	Name         string
	Status       string
	ImageFileID  string
	ChildTopics  []taxpkg.TreeNode
	Translations map[string]taxpkg.NodeTranslation
}

// UpdateCourseTopicInput carries partial-update data for a course topic.
type UpdateCourseTopicInput struct {
	Name               *string
	Status             *string
	ImageFileID        *string
	ChildTopics        *[]taxpkg.TreeNode
	Translations       map[string]taxpkg.NodeTranslation
	ExpectedRowVersion int64
}

// CreateCourseOutcomeInput carries data for creating a course outcome.
type CreateCourseOutcomeInput struct {
	ActorID          string
	ShortDescription string
	Description      []string
	Status           string
	ImageFileID      string
	Translations     map[string]OutcomeTranslation
}

// UpdateCourseOutcomeInput carries partial-update data for a course outcome.
type UpdateCourseOutcomeInput struct {
	ShortDescription   *string
	Description        *[]string
	Status             *string
	ImageFileID        *string
	Translations       map[string]OutcomeTranslation
	ExpectedRowVersion int64
}

// CreateCourseSkillInput carries data for creating a course skill.
type CreateCourseSkillInput struct {
	ActorID      string
	Name         string
	Status       string
	Children     []taxpkg.TreeNode
	Translations map[string]taxpkg.NodeTranslation
}

// UpdateCourseSkillInput carries partial-update data for a course skill.
type UpdateCourseSkillInput struct {
	Name               *string
	Status             *string
	Children           *[]taxpkg.TreeNode
	Translations       map[string]taxpkg.NodeTranslation
	ExpectedRowVersion int64
}

// CreateTagInput carries data for creating a new tag.
type CreateTagInput struct {
	ActorID      string
	Name         string
	Status       string
	Translations map[string]taxpkg.NodeTranslation
}

// UpdateTagInput carries partial-update data for a tag.
type UpdateTagInput struct {
	Name               *string
	Status             *string
	Translations       map[string]taxpkg.NodeTranslation
	ExpectedRowVersion int64
}

// CreateCourseLevelInput carries data for creating a new course level.
type CreateCourseLevelInput = CreateTagInput

// UpdateCourseLevelInput carries partial-update data for a course level.
type UpdateCourseLevelInput = UpdateTagInput
