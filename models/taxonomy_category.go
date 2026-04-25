package models

import (
	"mycourse-io-be/core/entities"
	"mycourse-io-be/dbschema"
)

type Category struct {
	entities.Category
}

func (Category) TableName() string { return dbschema.Taxonomy.Categories() }
