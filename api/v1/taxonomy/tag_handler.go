package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/logic/mapping"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listTags(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListTags, mapping.TagListHTTPPayload)
}

func createTag(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateTag, mapping.TagRowHTTPPayload)
}

func updateTag(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateTag, mapping.TagRowHTTPPayload)
}

func deleteTag(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteTag)
}
