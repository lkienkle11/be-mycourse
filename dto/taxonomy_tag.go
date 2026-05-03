package dto

type TagFilter struct {
	BaseFilter
	Status *string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
}

type CreateTagRequest struct {
	Name   string `json:"name" validate:"required,min=1,max=255"`
	Slug   string `json:"slug" validate:"required,min=1,max=255"`
	Status string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type UpdateTagRequest struct {
	Name   *string `json:"name" validate:"omitempty,min=1,max=255"`
	Slug   *string `json:"slug" validate:"omitempty,min=1,max=255"`
	Status *string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type TagResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedBy *uint  `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
