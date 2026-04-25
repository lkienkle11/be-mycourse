package models

import (
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/dbschema"
)

type CourseLevel struct {
	entities.CourseLevel
}

func (CourseLevel) TableName() string { return dbschema.Taxonomy.CourseLevels() }
