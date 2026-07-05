package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/utils"
)

// RegisterRoutes mounts instructor management APIs on rg.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	instructors := rg.Group("/instructors")
	instructors.GET("", utils.RoutePermission(pc, constants.AllPermissions.InstructorRosterRead), h.listRoster)
	instructors.GET("/roster-candidates", utils.RoutePermission(pc, constants.AllPermissions.InstructorRosterCreate), h.listRosterCandidates)
	instructors.POST("/bulk", utils.RoutePermission(pc, constants.AllPermissions.InstructorRosterCreate), h.addRosterBulk)
	instructors.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.InstructorRosterDelete), h.deleteRoster)

	instructors.GET("/:id/expertise/topics", utils.RoutePermission(pc, constants.AllPermissions.InstructorExpertiseRead), h.listExpertiseTopics)
	instructors.POST("/:id/expertise/topics", utils.RoutePermission(pc, constants.AllPermissions.InstructorExpertiseCreate), h.addExpertiseTopic)
	instructors.DELETE("/:id/expertise/topics/:topicRowId", utils.RoutePermission(pc, constants.AllPermissions.InstructorExpertiseDelete), h.deleteExpertiseTopicByRow)

	instructors.GET("/:id/expertise/skills", utils.RoutePermission(pc, constants.AllPermissions.InstructorExpertiseRead), h.listExpertiseSkills)
	instructors.POST("/:id/expertise/skills", utils.RoutePermission(pc, constants.AllPermissions.InstructorExpertiseCreate), h.addExpertiseSkill)
	instructors.DELETE("/:id/expertise/skills/:skillRowId", utils.RoutePermission(pc, constants.AllPermissions.InstructorExpertiseDelete), h.deleteExpertiseSkillByRow)

	apps := rg.Group("/instructor-applications")
	apps.GET("/me", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationCreate), h.getMyApplication)
	apps.PUT("/me", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationCreate), h.resubmitMyApplication)
	apps.POST("/contact-admin", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationCreate), h.contactAdmin)
	apps.GET("", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationRead), h.listApplications)
	apps.POST("", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationCreate), h.submitApplication)
	apps.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationRead), h.getApplication)
	apps.POST("/:id/approve", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationApprove), h.approveApplication)
	apps.POST("/:id/reject", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationReject), h.rejectApplication)
	apps.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationDelete), h.deleteApplication)

	profiles := rg.Group("/instructor-profiles")
	profiles.GET("", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileRead), h.listProfiles)
	profiles.GET("/me", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileRead), h.getProfileMe)
	profiles.GET("/:id", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileRead), h.getProfileByUser)
	profiles.POST("", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileCreate), h.upsertProfile)
	profiles.PATCH("/:id", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileUpdate), h.upsertProfile)
	profiles.DELETE("/:id", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileDelete), h.deleteProfile)

	tickets := rg.Group("/instructor-tickets")
	tickets.GET("", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationRead), h.listTickets)
	tickets.POST("", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationCreate), h.createTicket)
	tickets.POST("/:id/close", utils.RoutePermission(pc, constants.AllPermissions.InstructorTicketClose), h.closeTicket)
	tickets.GET("/:id/messages", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationRead), h.listTicketMessages)
	tickets.POST("/:id/messages", utils.RoutePermission(pc, constants.AllPermissions.InstructorApplicationCreate), h.addTicketMessage)

	stubs := rg.Group("/instructor-stubs")
	stubs.GET("/assignments", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileRead), h.comingSoon)
	stubs.GET("/activity-log", utils.RoutePermission(pc, constants.AllPermissions.InstructorProfileRead), h.comingSoon)
}
