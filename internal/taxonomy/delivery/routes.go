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
	levels.GET("/full", rp(constants.AllPermissions.CourseLevelRead), h.listCourseLevelsFull)
	levels.GET("", rp(constants.AllPermissions.CourseLevelRead), h.listCourseLevels)
	levels.POST("", rp(constants.AllPermissions.CourseLevelCreate), h.createCourseLevel)
	levels.PATCH("/:id", rp(constants.AllPermissions.CourseLevelUpdate), h.updateCourseLevel)
	levels.DELETE("/:id/hard", rp(constants.AllPermissions.CourseLevelDelete), h.hardDeleteCourseLevel)
	levels.DELETE("/:id", rp(constants.AllPermissions.CourseLevelDelete), h.deleteCourseLevel)

	topics := taxonomy.Group("/topics")
	topics.GET("/full", rp(constants.AllPermissions.TopicRead), h.listTopicsFull)
	topics.GET("", rp(constants.AllPermissions.TopicRead), h.listTopics)
	topics.POST("", rp(constants.AllPermissions.TopicCreate), h.createTopic)
	topics.PATCH("/:id", rp(constants.AllPermissions.TopicUpdate), h.updateTopic)
	topics.DELETE("/:id/hard", rp(constants.AllPermissions.TopicDelete), h.hardDeleteTopic)
	topics.DELETE("/:id", rp(constants.AllPermissions.TopicDelete), h.deleteTopic)

	outcomes := taxonomy.Group("/outcomes")
	outcomes.GET("/full", rp(constants.AllPermissions.CourseOutcomeRead), h.listCourseOutcomesFull)
	outcomes.GET("", rp(constants.AllPermissions.CourseOutcomeRead), h.listCourseOutcomes)
	outcomes.POST("", rp(constants.AllPermissions.CourseOutcomeCreate), h.createCourseOutcome)
	outcomes.PATCH("/:id", rp(constants.AllPermissions.CourseOutcomeUpdate), h.updateCourseOutcome)
	outcomes.DELETE("/:id/hard", rp(constants.AllPermissions.CourseOutcomeDelete), h.hardDeleteCourseOutcome)
	outcomes.DELETE("/:id", rp(constants.AllPermissions.CourseOutcomeDelete), h.deleteCourseOutcome)

	skills := taxonomy.Group("/skills")
	skills.GET("/full", rp(constants.AllPermissions.CourseSkillRead), h.listCourseSkillsFull)
	skills.GET("", rp(constants.AllPermissions.CourseSkillRead), h.listCourseSkills)
	skills.POST("", rp(constants.AllPermissions.CourseSkillCreate), h.createCourseSkill)
	skills.PATCH("/:id", rp(constants.AllPermissions.CourseSkillUpdate), h.updateCourseSkill)
	skills.DELETE("/:id/hard", rp(constants.AllPermissions.CourseSkillDelete), h.hardDeleteCourseSkill)
	skills.DELETE("/:id", rp(constants.AllPermissions.CourseSkillDelete), h.deleteCourseSkill)

	tags := taxonomy.Group("/tags")
	tags.GET("/full", rp(constants.AllPermissions.TagRead), h.listTagsFull)
	tags.GET("", rp(constants.AllPermissions.TagRead), h.listTags)
	tags.POST("", rp(constants.AllPermissions.TagCreate), h.createTag)
	tags.PATCH("/:id", rp(constants.AllPermissions.TagUpdate), h.updateTag)
	tags.DELETE("/:id/hard", rp(constants.AllPermissions.TagDelete), h.hardDeleteTag)
	tags.DELETE("/:id", rp(constants.AllPermissions.TagDelete), h.deleteTag)
}
