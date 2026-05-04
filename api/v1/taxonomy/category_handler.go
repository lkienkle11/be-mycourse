package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/mapping"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCategories(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListCategories, func(rows []entities.Category) any {
		return mapping.ToCategoryResponses(rows)
	})
}

func createCategory(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateCategory, func(row entities.Category) any {
		return mapping.ToCategoryResponse(row)
	})
}

func updateCategory(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateCategory, func(row entities.Category) any {
		return mapping.ToCategoryResponse(row)
	})
}

func deleteCategory(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteCategory)
}
