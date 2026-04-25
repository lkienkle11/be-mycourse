package models

import (
	"mycourse-io-be/core/entities"
	"mycourse-io-be/dbschema"
)

type CourseLevel struct {
	entities.CourseLevel
}

func (CourseLevel) TableName() string { return dbschema.Taxonomy.CourseLevels() }
