package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listTags(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListTags, func(rows []dto.TagResponse) any {
		return rows
	})
}

func createTag(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateTag, func(row dto.TagResponse) any {
		return row
	})
}

func updateTag(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateTag, func(row dto.TagResponse) any {
		return row
	})
}

func deleteTag(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteTag)
}
