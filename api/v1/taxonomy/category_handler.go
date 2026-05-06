package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCategories(c *gin.Context) {
	respondTaxonomyList(c, taxonomyservice.ListCategories, func(rows []dto.CategoryResponse) any {
		return rows
	})
}

func createCategory(c *gin.Context) {
	respondTaxonomyCreate(c, taxonomyservice.CreateCategory, func(row dto.CategoryResponse) any {
		return row
	})
}

func updateCategory(c *gin.Context) {
	respondTaxonomyUpdate(c, taxonomyservice.UpdateCategory, func(row dto.CategoryResponse) any {
		return row
	})
}

func deleteCategory(c *gin.Context) {
	respondTaxonomyDelete(c, taxonomyservice.DeleteCategory)
}
