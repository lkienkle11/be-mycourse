package taxonomy

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	taxonomy := rg.Group("/taxonomy")

	levels := taxonomy.Group("/levels")
	levels.GET("", middleware.RequirePermission(constants.AllPermissions.CourseLevelRead), listCourseLevels)
	levels.POST("", middleware.RequirePermission(constants.AllPermissions.CourseLevelCreate), createCourseLevel)
	levels.PATCH("/:id", middleware.RequirePermission(constants.AllPermissions.CourseLevelUpdate), updateCourseLevel)
	levels.DELETE("/:id", middleware.RequirePermission(constants.AllPermissions.CourseLevelDelete), deleteCourseLevel)

	categories := taxonomy.Group("/categories")
	categories.GET("", middleware.RequirePermission(constants.AllPermissions.CategoryRead), listCategories)
	categories.POST("", middleware.RequirePermission(constants.AllPermissions.CategoryCreate), createCategory)
	categories.PATCH("/:id", middleware.RequirePermission(constants.AllPermissions.CategoryUpdate), updateCategory)
	categories.DELETE("/:id", middleware.RequirePermission(constants.AllPermissions.CategoryDelete), deleteCategory)

	tags := taxonomy.Group("/tags")
	tags.GET("", middleware.RequirePermission(constants.AllPermissions.TagRead), listTags)
	tags.POST("", middleware.RequirePermission(constants.AllPermissions.TagCreate), createTag)
	tags.PATCH("/:id", middleware.RequirePermission(constants.AllPermissions.TagUpdate), updateTag)
	tags.DELETE("/:id", middleware.RequirePermission(constants.AllPermissions.TagDelete), deleteTag)
}
