package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/utils"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	courses := rg.Group("/courses")
	courses.GET("/my", utils.RoutePermission(pc, constants.AllPermissions.CourseInstructorRead), h.listEditableCourses)
	courses.POST("", utils.RoutePermission(pc, constants.AllPermissions.CourseCreate), h.createCourse)
	courses.GET("/:courseId", utils.RoutePermission(pc, constants.AllPermissions.CourseInstructorRead), h.getCourseDetail)
	courses.POST("/:courseId/draft/prepare", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.prepareDraft)
	courses.PATCH("/:courseId/basic-info", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.updateBasicInfo)
	courses.DELETE("/:courseId", utils.RoutePermission(pc, constants.AllPermissions.CourseDelete), h.deleteCourse)

	courses.GET("/:courseId/collaborators", utils.RoutePermission(pc, constants.AllPermissions.CourseInstructorRead), h.listCollaborators)
	courses.POST("/:courseId/collaborators", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.addCollaborator)
	courses.DELETE("/:courseId/collaborators/:userId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.removeCollaborator)

	courses.POST("/:courseId/sections", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.createSection)
	courses.PATCH("/:courseId/sections/:sectionId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.updateSection)
	courses.DELETE("/:courseId/sections/:sectionId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.deleteSection)
	courses.POST("/:courseId/sections/reorder", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.reorderSections)

	courses.POST("/:courseId/lessons", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.createLesson)
	courses.PATCH("/:courseId/lessons/:lessonId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.updateLesson)
	courses.DELETE("/:courseId/lessons/:lessonId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.deleteLesson)
	courses.POST("/:courseId/sections/:sectionId/lessons/reorder", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.reorderLessons)

	courses.POST("/:courseId/sub-lessons", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.createSubLesson)
	courses.PATCH("/:courseId/sub-lessons/:subLessonId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.updateSubLesson)
	courses.DELETE("/:courseId/sub-lessons/:subLessonId", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.deleteSubLesson)
	courses.POST("/:courseId/lessons/:lessonId/sub-lessons/reorder", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.reorderSubLessons)

	courses.POST("/:courseId/leases/acquire", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.acquireLease)
	courses.POST("/:courseId/leases/heartbeat", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.heartbeatLease)
	courses.POST("/:courseId/leases/release", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.releaseLease)

	courses.POST("/:courseId/submit-review", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.submitForReview)
	courses.POST("/:courseId/reopen-draft", utils.RoutePermission(pc, constants.AllPermissions.CourseUpdate), h.reopenDraft)
	courses.GET("/:courseId/review-history", utils.RoutePermission(pc, constants.AllPermissions.CourseInstructorRead), h.listReviewHistory)

	reviews := rg.Group("/course-reviews")
	reviews.GET("/pending", utils.RoutePermission(pc, constants.AllPermissions.CourseReviewRead), h.listPendingReviews)
	reviews.POST("/:courseId/approve", utils.RoutePermission(pc, constants.AllPermissions.CourseReviewApprove), h.approveDraft)
	reviews.POST("/:courseId/reject", utils.RoutePermission(pc, constants.AllPermissions.CourseReviewReject), h.rejectDraft)

	registerCourseAdminRoutes(rg, h, pc)

	learner := rg.Group("/learner-courses")
	learner.GET("", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.listPublishedCourses)
	learner.GET("/:courseId", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.getLearningCourse)
	learner.POST("/:courseId/enroll", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.enroll)
	learner.GET("/:courseId/progress", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.getProgress)
	learner.POST("/:courseId/progress", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.saveProgress)
}

func registerCourseAdminRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	adminCourses := rg.Group("/course-admin")
	adminCourses.GET("/courses", utils.RoutePermission(pc, constants.AllPermissions.CourseCatalogRead), h.listAdminCourses)
	adminCourses.GET("/courses/trash", utils.RoutePermission(pc, constants.AllPermissions.CourseTrashRead), h.listTrashedCourses)
	adminCourses.POST("/courses/:courseId/trash", utils.RoutePermission(pc, constants.AllPermissions.CourseCatalogTrash), h.trashCourse)
	adminCourses.POST("/courses/:courseId/restore", utils.RoutePermission(pc, constants.AllPermissions.CourseTrashRestore), h.restoreCourse)
	adminCourses.DELETE("/courses/:courseId/permanent", utils.RoutePermission(pc, constants.AllPermissions.CourseTrashDelete), h.permanentDeleteCourse)
}
