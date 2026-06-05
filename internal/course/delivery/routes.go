package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	rp := func(actions ...string) gin.HandlerFunc { return middleware.RequirePermission(pc, actions...) }

	courses := rg.Group("/courses")
	courses.GET("/my", rp(constants.AllPermissions.CourseInstructorRead), h.listEditableCourses)
	courses.POST("", rp(constants.AllPermissions.CourseCreate), h.createCourse)
	courses.GET("/:courseId", rp(constants.AllPermissions.CourseInstructorRead), h.getCourseDetail)
	courses.POST("/:courseId/draft/prepare", rp(constants.AllPermissions.CourseUpdate), h.prepareDraft)
	courses.PATCH("/:courseId/basic-info", rp(constants.AllPermissions.CourseUpdate), h.updateBasicInfo)
	courses.DELETE("/:courseId", rp(constants.AllPermissions.CourseDelete), h.deleteCourse)

	courses.GET("/:courseId/collaborators", rp(constants.AllPermissions.CourseInstructorRead), h.listCollaborators)
	courses.POST("/:courseId/collaborators", rp(constants.AllPermissions.CourseUpdate), h.addCollaborator)
	courses.DELETE("/:courseId/collaborators/:userId", rp(constants.AllPermissions.CourseUpdate), h.removeCollaborator)

	courses.POST("/:courseId/sections", rp(constants.AllPermissions.CourseUpdate), h.createSection)
	courses.PATCH("/:courseId/sections/:sectionId", rp(constants.AllPermissions.CourseUpdate), h.updateSection)
	courses.DELETE("/:courseId/sections/:sectionId", rp(constants.AllPermissions.CourseUpdate), h.deleteSection)
	courses.POST("/:courseId/sections/reorder", rp(constants.AllPermissions.CourseUpdate), h.reorderSections)

	courses.POST("/:courseId/lessons", rp(constants.AllPermissions.CourseUpdate), h.createLesson)
	courses.PATCH("/:courseId/lessons/:lessonId", rp(constants.AllPermissions.CourseUpdate), h.updateLesson)
	courses.DELETE("/:courseId/lessons/:lessonId", rp(constants.AllPermissions.CourseUpdate), h.deleteLesson)
	courses.POST("/:courseId/sections/:sectionId/lessons/reorder", rp(constants.AllPermissions.CourseUpdate), h.reorderLessons)

	courses.POST("/:courseId/sub-lessons", rp(constants.AllPermissions.CourseUpdate), h.createSubLesson)
	courses.PATCH("/:courseId/sub-lessons/:subLessonId", rp(constants.AllPermissions.CourseUpdate), h.updateSubLesson)
	courses.DELETE("/:courseId/sub-lessons/:subLessonId", rp(constants.AllPermissions.CourseUpdate), h.deleteSubLesson)
	courses.POST("/:courseId/lessons/:lessonId/sub-lessons/reorder", rp(constants.AllPermissions.CourseUpdate), h.reorderSubLessons)

	courses.POST("/:courseId/leases/acquire", rp(constants.AllPermissions.CourseUpdate), h.acquireLease)
	courses.POST("/:courseId/leases/heartbeat", rp(constants.AllPermissions.CourseUpdate), h.heartbeatLease)
	courses.POST("/:courseId/leases/release", rp(constants.AllPermissions.CourseUpdate), h.releaseLease)

	courses.POST("/:courseId/submit-review", rp(constants.AllPermissions.CourseUpdate), h.submitForReview)
	courses.POST("/:courseId/reopen-draft", rp(constants.AllPermissions.CourseUpdate), h.reopenDraft)

	reviews := rg.Group("/course-reviews")
	reviews.GET("/pending", rp(constants.AllPermissions.AdminModify), h.listPendingReviews)
	reviews.POST("/:courseId/approve", rp(constants.AllPermissions.AdminModify), h.approveDraft)
	reviews.POST("/:courseId/reject", rp(constants.AllPermissions.AdminModify), h.rejectDraft)

	learner := rg.Group("/learner-courses")
	learner.GET("", rp(constants.AllPermissions.CourseRead), h.listPublishedCourses)
	learner.GET("/:courseId", rp(constants.AllPermissions.CourseRead), h.getLearningCourse)
	learner.POST("/:courseId/enroll", rp(constants.AllPermissions.CourseRead), h.enroll)
	learner.GET("/:courseId/progress", rp(constants.AllPermissions.CourseRead), h.getProgress)
	learner.POST("/:courseId/progress", rp(constants.AllPermissions.CourseRead), h.saveProgress)
}
