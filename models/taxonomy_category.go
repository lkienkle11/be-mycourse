package models

import (
	"mycourse-io-be/dbschema"
	"mycourse-io-be/pkg/entities"
)

type Category struct {
	entities.Category
	ImageFile *MediaFile `gorm:"foreignKey:ImageFileID;references:ID"`
}

func (Category) TableName() string { return dbschema.Taxonomy.Categories() }
