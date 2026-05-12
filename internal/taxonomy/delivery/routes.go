package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
)

// RegisterRoutes mounts all taxonomy endpoints on rg.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	rp := func(actions ...string) gin.HandlerFunc { return middleware.RequirePermission(pc, actions...) }

	taxonomy := rg.Group("/taxonomy")

	levels := taxonomy.Group("/levels")
	levels.GET("", rp(constants.AllPermissions.CourseLevelRead), h.listCourseLevels)
	levels.POST("", rp(constants.AllPermissions.CourseLevelCreate), h.createCourseLevel)
	levels.PATCH("/:id", rp(constants.AllPermissions.CourseLevelUpdate), h.updateCourseLevel)
	levels.DELETE("/:id", rp(constants.AllPermissions.CourseLevelDelete), h.deleteCourseLevel)

	categories := taxonomy.Group("/categories")
	categories.GET("", rp(constants.AllPermissions.CategoryRead), h.listCategories)
	categories.POST("", rp(constants.AllPermissions.CategoryCreate), h.createCategory)
	categories.PATCH("/:id", rp(constants.AllPermissions.CategoryUpdate), h.updateCategory)
	categories.DELETE("/:id", rp(constants.AllPermissions.CategoryDelete), h.deleteCategory)

	tags := taxonomy.Group("/tags")
	tags.GET("", rp(constants.AllPermissions.TagRead), h.listTags)
	tags.POST("", rp(constants.AllPermissions.TagCreate), h.createTag)
	tags.PATCH("/:id", rp(constants.AllPermissions.TagUpdate), h.updateTag)
	tags.DELETE("/:id", rp(constants.AllPermissions.TagDelete), h.deleteTag)
}
