package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCourseLevels(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListCourseLevels, func(rows []dto.CourseLevelResponse) any {
		return rows
	})
}

func createCourseLevel(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateCourseLevel, func(row dto.CourseLevelResponse) any {
		return row
	})
}

func updateCourseLevel(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateCourseLevel, func(row dto.CourseLevelResponse) any {
		return row
	})
}

func deleteCourseLevel(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteCourseLevel)
}
