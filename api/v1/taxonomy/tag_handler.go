package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/mapping"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listTags(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListTags, func(rows []entities.Tag) any {
		return mapping.ToTagResponses(rows)
	})
}

func createTag(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateTag, func(row entities.Tag) any {
		return mapping.ToTagResponse(row)
	})
}

func updateTag(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateTag, func(row entities.Tag) any {
		return mapping.ToTagResponse(row)
	})
}

func deleteTag(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteTag)
}
