// Package delivery contains the TAXONOMY bounded-context HTTP delivery layer.
package delivery

import taxpkg "mycourse-io-be/internal/shared/taxonomy"

// TaxonomyBaseFilter is the shared pagination/search query params for taxonomy list endpoints.
type TaxonomyBaseFilter struct {
	Page        int     `form:"page"`
	PerPage     int     `form:"per_page"`
	Status      *string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	SortBy      string  `form:"sort_by"`
	SortDesc    bool    `form:"sort_desc"`
	SearchBy    string  `form:"search_by"`
	SearchValue string  `form:"search_value"`
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
	Name        string            `json:"name" validate:"required,min=1,max=255"`
	ImageFileID string            `json:"image_file_id" validate:"omitempty,uuid"`
	ChildTopics []taxpkg.TreeNode `json:"child_topics"`
	Status      string            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateCourseTopicRequest is the JSON body for updating a course topic.
type UpdateCourseTopicRequest struct {
	Name        *string            `json:"name" validate:"omitempty,min=1,max=255"`
	ImageFileID *string            `json:"image_file_id" validate:"omitempty,uuid"`
	ChildTopics *[]taxpkg.TreeNode `json:"child_topics"`
	Status      *string            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CreateCourseOutcomeRequest is the JSON body for creating a course outcome.
type CreateCourseOutcomeRequest struct {
	ShortDescription string   `json:"short_description" validate:"required,min=1,max=100"`
	Description      []string `json:"description"`
	ImageFileID      string   `json:"image_file_id" validate:"omitempty,uuid"`
	Status           string   `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateCourseOutcomeRequest is the JSON body for updating a course outcome.
type UpdateCourseOutcomeRequest struct {
	ShortDescription *string   `json:"short_description" validate:"omitempty,min=1,max=100"`
	Description      *[]string `json:"description"`
	ImageFileID      *string   `json:"image_file_id" validate:"omitempty,uuid"`
	Status           *string   `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CreateCourseSkillRequest is the JSON body for creating a course skill.
type CreateCourseSkillRequest struct {
	Name     string            `json:"name" validate:"required,min=1,max=255"`
	Children []taxpkg.TreeNode `json:"children"`
	Status   string            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateCourseSkillRequest is the JSON body for updating a course skill.
type UpdateCourseSkillRequest struct {
	Name     *string            `json:"name" validate:"omitempty,min=1,max=255"`
	Children *[]taxpkg.TreeNode `json:"children"`
	Status   *string            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CreateTagRequest is the JSON body for creating a tag.
type CreateTagRequest struct {
	Name   string `json:"name" validate:"required,min=1,max=255"`
	Status string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateTagRequest is the JSON body for updating a tag.
type UpdateTagRequest struct {
	Name   *string `json:"name" validate:"omitempty,min=1,max=255"`
	Status *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CreateCourseLevelRequest is the JSON body for creating a course level.
type CreateCourseLevelRequest = CreateTagRequest

// UpdateCourseLevelRequest is the JSON body for updating a course level.
type UpdateCourseLevelRequest = UpdateTagRequest

// CourseTopicResponse is the JSON response for a course topic.
type CourseTopicResponse struct {
	ID           uint              `json:"id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	ImageFileID  string            `json:"image_file_id,omitempty"`
	ImageFileURL string            `json:"image_file_url,omitempty"`
	ChildTopics  []taxpkg.TreeNode `json:"child_topics"`
	Status       string            `json:"status"`
	CreatedBy    *uint             `json:"created_by,omitempty"`
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
}

// CourseOutcomeResponse is the JSON response for a course outcome.
type CourseOutcomeResponse struct {
	ID               uint     `json:"id"`
	ShortDescription string   `json:"short_description"`
	Description      []string `json:"description"`
	ImageFileID      string   `json:"image_file_id,omitempty"`
	ImageFileURL     string   `json:"image_file_url,omitempty"`
	Status           string   `json:"status"`
	CreatedBy        *uint    `json:"created_by,omitempty"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
}

// CourseSkillResponse is the JSON response for a course skill.
type CourseSkillResponse struct {
	ID        uint              `json:"id"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Children  []taxpkg.TreeNode `json:"children"`
	Status    string            `json:"status"`
	CreatedBy *uint             `json:"created_by,omitempty"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

// SlugStatusResponse is the shared JSON shape for simple slug+status taxonomy resources.
type SlugStatusResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedBy *uint  `json:"created_by,omitempty"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// TagResponse is the JSON response for a tag.
type TagResponse = SlugStatusResponse

// CourseLevelResponse is the JSON response for a course level.
type CourseLevelResponse = SlugStatusResponse
