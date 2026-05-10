package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/logic/mapping"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCourseLevels(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListCourseLevels, mapping.CourseLevelListHTTPPayload)
}

func createCourseLevel(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateCourseLevel, mapping.CourseLevelRowHTTPPayload)
}

func updateCourseLevel(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateCourseLevel, mapping.CourseLevelRowHTTPPayload)
}

func deleteCourseLevel(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteCourseLevel)
}
