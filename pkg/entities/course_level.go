package entities

import (
	"time"

	"mycourse-io-be/constants"
)

type CourseLevel struct {
	ID        uint                     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string                   `gorm:"size:255;not null" json:"name"`
	Slug      string                   `gorm:"size:255;not null;uniqueIndex" json:"slug"`
	Status    constants.TaxonomyStatus `gorm:"type:taxonomy_status;not null;default:'ACTIVE'" json:"status"`
	CreatedBy *uint                    `gorm:"column:created_by" json:"created_by,omitempty"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}
