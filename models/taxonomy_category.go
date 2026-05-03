package models

import (
	"mycourse-io-be/dbschema"
	"mycourse-io-be/pkg/entities"
)

type Category struct {
	entities.Category
}

func (Category) TableName() string { return dbschema.Taxonomy.Categories() }
