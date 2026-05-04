package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/mapping"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCourseLevels(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListCourseLevels, func(rows []entities.CourseLevel) any {
		return mapping.ToCourseLevelResponses(rows)
	})
}

func createCourseLevel(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateCourseLevel, func(row entities.CourseLevel) any {
		return mapping.ToCourseLevelResponse(row)
	})
}

func updateCourseLevel(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateCourseLevel, func(row entities.CourseLevel) any {
		return mapping.ToCourseLevelResponse(row)
	})
}

func deleteCourseLevel(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteCourseLevel)
}
