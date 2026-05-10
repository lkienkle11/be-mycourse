package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/logic/mapping"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCategories(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListCategories, mapping.CategoryListHTTPPayload)
}

func createCategory(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateCategory, mapping.CategoryRowHTTPPayload)
}

func updateCategory(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateCategory, mapping.CategoryRowHTTPPayload)
}

func deleteCategory(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteCategory)
}
