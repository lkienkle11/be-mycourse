// Package delivery contains the TAXONOMY bounded-context HTTP delivery layer.
package delivery

// TaxonomyBaseFilter is the shared pagination/search query params for taxonomy list endpoints.
type TaxonomyBaseFilter struct {
	Page     int     `form:"page"`
	PerPage  int     `form:"per_page"`
	Status   *string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	SortBy   string  `form:"sort_by"`
	SortDesc bool    `form:"sort_desc"`
	Search   string  `form:"search"`
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

// CreateCategoryRequest is the JSON body for creating a category.
type CreateCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	ImageFileID string `json:"image_file_id" validate:"required,uuid"`
	Status      string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateCategoryRequest is the JSON body for updating a category.
type UpdateCategoryRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=255"`
	Slug        *string `json:"slug" validate:"omitempty,min=1,max=255"`
	ImageFileID *string `json:"image_file_id" validate:"omitempty,uuid"`
	Status      *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CreateTagRequest is the JSON body for creating a tag.
type CreateTagRequest struct {
	Name   string `json:"name" validate:"required,min=1,max=255"`
	Slug   string `json:"slug" validate:"required,min=1,max=255"`
	Status string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateTagRequest is the JSON body for updating a tag.
type UpdateTagRequest struct {
	Name   *string `json:"name" validate:"omitempty,min=1,max=255"`
	Slug   *string `json:"slug" validate:"omitempty,min=1,max=255"`
	Status *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CreateCourseLevelRequest is the JSON body for creating a course level.
type CreateCourseLevelRequest struct {
	Name   string `json:"name" validate:"required,min=1,max=255"`
	Slug   string `json:"slug" validate:"required,min=1,max=255"`
	Status string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// UpdateCourseLevelRequest is the JSON body for updating a course level.
type UpdateCourseLevelRequest struct {
	Name   *string `json:"name" validate:"omitempty,min=1,max=255"`
	Slug   *string `json:"slug" validate:"omitempty,min=1,max=255"`
	Status *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

// CategoryResponse is the JSON response for a category.
type CategoryResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ImageFileID string `json:"image_file_id,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	Status      string `json:"status"`
	CreatedBy   *uint  `json:"created_by,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TagResponse is the JSON response for a tag.
type TagResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedBy *uint  `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CourseLevelResponse is the JSON response for a course level.
type CourseLevelResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedBy *uint  `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
