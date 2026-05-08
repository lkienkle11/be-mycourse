package entities

import (
	"time"
)

type Category struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	ImageFileID *string   `json:"-"`
	Status      string    `json:"status"`
	CreatedBy   *uint     `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
