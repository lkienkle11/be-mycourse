package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/utils"
)

// RegisterRoutes mounts all taxonomy endpoints on rg.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	taxonomy := rg.Group("/taxonomy")

	levels := taxonomy.Group("/levels")
	levels.GET("/full", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelRead), h.listCourseLevelsFull)
	levels.GET("", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelRead), h.listCourseLevels)
	levels.POST("", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelCreate), h.createCourseLevel)
	levels.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelRead), h.getCourseLevel)
	levels.PATCH("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelUpdate), h.updateCourseLevel)
	levels.DELETE("/:id/hard", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelDelete), h.hardDeleteCourseLevel)
	levels.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseLevelDelete), h.deleteCourseLevel)

	topics := taxonomy.Group("/topics")
	topics.GET("/full", utils.RoutePermission(pc, constants.AllPermissions.TopicRead), h.listTopicsFull)
	topics.GET("", utils.RoutePermission(pc, constants.AllPermissions.TopicRead), h.listTopics)
	topics.POST("", utils.RoutePermission(pc, constants.AllPermissions.TopicCreate), h.createTopic)
	topics.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.TopicRead), h.getTopic)
	topics.PATCH("/:id", utils.RoutePermission(pc, constants.AllPermissions.TopicUpdate), h.updateTopic)
	topics.DELETE("/:id/hard", utils.RoutePermission(pc, constants.AllPermissions.TopicDelete), h.hardDeleteTopic)
	topics.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.TopicDelete), h.deleteTopic)

	outcomes := taxonomy.Group("/outcomes")
	outcomes.GET("/full", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeRead), h.listCourseOutcomesFull)
	outcomes.GET("", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeRead), h.listCourseOutcomes)
	outcomes.POST("", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeCreate), h.createCourseOutcome)
	outcomes.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeRead), h.getCourseOutcome)
	outcomes.PATCH("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeUpdate), h.updateCourseOutcome)
	outcomes.DELETE("/:id/hard", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeDelete), h.hardDeleteCourseOutcome)
	outcomes.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseOutcomeDelete), h.deleteCourseOutcome)

	skills := taxonomy.Group("/skills")
	skills.GET("/full", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillRead), h.listCourseSkillsFull)
	skills.GET("", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillRead), h.listCourseSkills)
	skills.POST("", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillCreate), h.createCourseSkill)
	skills.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillRead), h.getCourseSkill)
	skills.PATCH("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillUpdate), h.updateCourseSkill)
	skills.DELETE("/:id/hard", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillDelete), h.hardDeleteCourseSkill)
	skills.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.CourseSkillDelete), h.deleteCourseSkill)

	registerTagRoutes(taxonomy, h, pc)
}

func registerTagRoutes(taxonomy *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	tags := taxonomy.Group("/tags")
	tags.GET("/full", utils.RoutePermission(pc, constants.AllPermissions.TagRead), h.listTagsFull)
	tags.GET("", utils.RoutePermission(pc, constants.AllPermissions.TagRead), h.listTags)
	tags.POST("", utils.RoutePermission(pc, constants.AllPermissions.TagCreate), h.createTag)
	tags.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.TagRead), h.getTag)
	tags.PATCH("/:id", utils.RoutePermission(pc, constants.AllPermissions.TagUpdate), h.updateTag)
	tags.DELETE("/:id/hard", utils.RoutePermission(pc, constants.AllPermissions.TagDelete), h.hardDeleteTag)
	tags.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.TagDelete), h.deleteTag)
}
