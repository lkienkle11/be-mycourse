package models

import (
	"mycourse-io-be/dbschema"
	"mycourse-io-be/pkg/entities"
)

type CourseLevel struct {
	entities.CourseLevel
}

func (CourseLevel) TableName() string { return dbschema.Taxonomy.CourseLevels() }
