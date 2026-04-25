package models

import (
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/dbschema"
)

type Tag struct {
	entities.Tag
}

func (Tag) TableName() string { return dbschema.Taxonomy.Tags() }
