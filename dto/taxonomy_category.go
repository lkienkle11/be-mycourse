package dto

type CategoryFilter struct {
	BaseFilter
	Status *string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
}

type CreateCategoryRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Slug     string `json:"slug" validate:"required,min=1,max=255"`
	ImageURL string `json:"image_url" validate:"required,max=512"`
	Status   string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=1,max=255"`
	Slug     *string `json:"slug" validate:"omitempty,min=1,max=255"`
	ImageURL *string `json:"image_url" validate:"omitempty,max=512"`
	Status   *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type CategoryResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ImageURL  string `json:"image_url"`
	Status    string `json:"status"`
	CreatedBy *uint  `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
