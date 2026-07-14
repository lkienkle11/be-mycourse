// Package delivery contains the TAXONOMY bounded-context HTTP delivery layer.
package delivery

import taxpkg "mycourse-io-be/internal/shared/taxonomy"

// TaxonomyBaseFilter is the shared pagination/search query params for taxonomy list endpoints.
type TaxonomyBaseFilter struct {
	Page          int     `form:"page"`
	PerPage       int     `form:"per_page"`
	Status        *string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	SortBy        string  `form:"sort_by"`
	SortDesc      bool    `form:"sort_desc"`
	SearchBy      string  `form:"search_by"`
	SearchValue   string  `form:"search_value"`
	IncludeImages *bool   `form:"include_images"`
	Locale        string  `form:"locale"`
}

// TaxonomyGetQuery is the query for GET /:id (locale + optional view=edit).
type TaxonomyGetQuery struct {
	Locale string `form:"locale"`
	View   string `form:"view"`
}

func (f TaxonomyBaseFilter) getPage() int {
	if f.Page < 1 {
		return 1
	}
	return f.Page
}

func (f TaxonomyBaseFilter) getPerPage() int {
	if f.PerPage < 1 {
		return 20
	}
	if f.PerPage > 100 {
		return 100
	}
	return f.PerPage
}

// CreateCourseTopicRequest is the JSON body for creating a course topic.
type CreateCourseTopicRequest struct {
	Name         string                            `json:"name" validate:"omitempty,min=1,max=255"`
	ImageFileID  string                            `json:"image_file_id" validate:"omitempty,uuid"`
	ChildTopics  []taxpkg.TreeNode                 `json:"child_topics"`
	Status       string                            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
}

// UpdateCourseTopicRequest is the JSON body for updating a course topic.
type UpdateCourseTopicRequest struct {
	Name               *string                           `json:"name" validate:"omitempty,min=1,max=255"`
	ImageFileID        *string                           `json:"image_file_id" validate:"omitempty,uuid"`
	ChildTopics        *[]taxpkg.TreeNode                `json:"child_topics"`
	Status             *string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]taxpkg.NodeTranslation `json:"translations"`
	ExpectedRowVersion int64                             `json:"expected_row_version" validate:"required,min=1"`
}

// OutcomeTranslationDTO is the per-locale outcome translation payload.
type OutcomeTranslationDTO struct {
	ShortDescription string   `json:"short_description"`
	Description      []string `json:"description"`
}

// CreateCourseOutcomeRequest is the JSON body for creating a course outcome.
type CreateCourseOutcomeRequest struct {
	ShortDescription string                           `json:"short_description" validate:"omitempty,min=1,max=100"`
	Description      []string                         `json:"description"`
	ImageFileID      string                           `json:"image_file_id" validate:"omitempty,uuid"`
	Status           string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations     map[string]OutcomeTranslationDTO `json:"translations"`
}

// UpdateCourseOutcomeRequest is the JSON body for updating a course outcome.
type UpdateCourseOutcomeRequest struct {
	ShortDescription   *string                          `json:"short_description" validate:"omitempty,min=1,max=100"`
	Description        *[]string                        `json:"description"`
	ImageFileID        *string                          `json:"image_file_id" validate:"omitempty,uuid"`
	Status             *string                          `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]OutcomeTranslationDTO `json:"translations"`
	ExpectedRowVersion int64                            `json:"expected_row_version" validate:"required,min=1"`
}

// CreateCourseSkillRequest is the JSON body for creating a course skill.
type CreateCourseSkillRequest struct {
	Name         string                            `json:"name" validate:"omitempty,min=1,max=255"`
	Children     []taxpkg.TreeNode                 `json:"children"`
	Status       string                            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
}

// UpdateCourseSkillRequest is the JSON body for updating a course skill.
type UpdateCourseSkillRequest struct {
	Name               *string                           `json:"name" validate:"omitempty,min=1,max=255"`
	Children           *[]taxpkg.TreeNode                `json:"children"`
	Status             *string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]taxpkg.NodeTranslation `json:"translations"`
	ExpectedRowVersion int64                             `json:"expected_row_version" validate:"required,min=1"`
}

// CreateTagRequest is the JSON body for creating a tag.
type CreateTagRequest struct {
	Name         string                            `json:"name" validate:"omitempty,min=1,max=255"`
	Status       string                            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
}

// UpdateTagRequest is the JSON body for updating a tag.
type UpdateTagRequest struct {
	Name               *string                           `json:"name" validate:"omitempty,min=1,max=255"`
	Status             *string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]taxpkg.NodeTranslation `json:"translations"`
	ExpectedRowVersion int64                             `json:"expected_row_version" validate:"required,min=1"`
}

// CreateCourseLevelRequest is the JSON body for creating a course level.
type CreateCourseLevelRequest = CreateTagRequest

// UpdateCourseLevelRequest is the JSON body for updating a course level.
type UpdateCourseLevelRequest = UpdateTagRequest

// CourseTopicResponse is the JSON response for a course topic.
type CourseTopicResponse struct {
	ID               string                            `json:"id"`
	Name             string                            `json:"name"`
	Slug             string                            `json:"slug"`
	ImageFileID      string                            `json:"image_file_id,omitempty"`
	ImageFileURL     string                            `json:"image_file_url,omitempty"`
	ChildTopics      []taxpkg.TreeNode                 `json:"child_topics"`
	Status           string                            `json:"status"`
	ResolvedLocale   string                            `json:"resolved_locale,omitempty"`
	AvailableLocales []string                          `json:"available_locales,omitempty"`
	Translations     map[string]taxpkg.NodeTranslation `json:"translations,omitempty"`
	RowVersion       int64                             `json:"row_version,omitempty"`
	CreatedBy        *string                           `json:"created_by,omitempty"`
	CreatedAt        int64                             `json:"created_at"`
	UpdatedAt        int64                             `json:"updated_at"`
}

// CourseOutcomeResponse is the JSON response for a course outcome.
type CourseOutcomeResponse struct {
	ID               string                           `json:"id"`
	ShortDescription string                           `json:"short_description"`
	Description      []string                         `json:"description"`
	ImageFileID      string                           `json:"image_file_id,omitempty"`
	ImageFileURL     string                           `json:"image_file_url,omitempty"`
	Status           string                           `json:"status"`
	ResolvedLocale   string                           `json:"resolved_locale,omitempty"`
	AvailableLocales []string                         `json:"available_locales,omitempty"`
	Translations     map[string]OutcomeTranslationDTO `json:"translations,omitempty"`
	RowVersion       int64                            `json:"row_version,omitempty"`
	CreatedBy        *string                          `json:"created_by,omitempty"`
	CreatedAt        int64                            `json:"created_at"`
	UpdatedAt        int64                            `json:"updated_at"`
}

// CourseSkillResponse is the JSON response for a course skill.
type CourseSkillResponse struct {
	ID               string                            `json:"id"`
	Name             string                            `json:"name"`
	Slug             string                            `json:"slug"`
	Children         []taxpkg.TreeNode                 `json:"children"`
	Status           string                            `json:"status"`
	ResolvedLocale   string                            `json:"resolved_locale,omitempty"`
	AvailableLocales []string                          `json:"available_locales,omitempty"`
	Translations     map[string]taxpkg.NodeTranslation `json:"translations,omitempty"`
	RowVersion       int64                             `json:"row_version,omitempty"`
	CreatedBy        *string                           `json:"created_by,omitempty"`
	CreatedAt        int64                             `json:"created_at"`
	UpdatedAt        int64                             `json:"updated_at"`
}

// SlugStatusResponse is the shared JSON shape for simple slug+status taxonomy resources.
type SlugStatusResponse struct {
	ID               string                            `json:"id"`
	Name             string                            `json:"name"`
	Slug             string                            `json:"slug"`
	Status           string                            `json:"status"`
	ResolvedLocale   string                            `json:"resolved_locale,omitempty"`
	AvailableLocales []string                          `json:"available_locales,omitempty"`
	Translations     map[string]taxpkg.NodeTranslation `json:"translations,omitempty"`
	RowVersion       int64                             `json:"row_version,omitempty"`
	CreatedBy        *string                           `json:"created_by,omitempty"`
	CreatedAt        int64                             `json:"created_at"`
	UpdatedAt        int64                             `json:"updated_at"`
}

// TagResponse is the JSON response for a tag.
type TagResponse = SlugStatusResponse

// CourseLevelResponse is the JSON response for a course level.
type CourseLevelResponse = SlugStatusResponse
