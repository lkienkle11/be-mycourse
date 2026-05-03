package models

import (
	"mycourse-io-be/dbschema"
	"mycourse-io-be/pkg/entities"
)

type Tag struct {
	entities.Tag
}

func (Tag) TableName() string { return dbschema.Taxonomy.Tags() }
