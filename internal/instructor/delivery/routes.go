package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
)

// RegisterRoutes mounts instructor management APIs on rg.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	rp := func(actions ...string) gin.HandlerFunc { return middleware.RequirePermission(pc, actions...) }

	instructors := rg.Group("/instructors")
	instructors.GET("", rp(constants.AllPermissions.InstructorRosterRead), h.listRoster)
	instructors.POST("", rp(constants.AllPermissions.InstructorRosterCreate), h.addRoster)
	instructors.DELETE("/:id", rp(constants.AllPermissions.InstructorRosterDelete), h.deleteRoster)

	instructors.GET("/:id/expertise/topics", rp(constants.AllPermissions.InstructorExpertiseRead), h.listExpertiseTopics)
	instructors.POST("/:id/expertise/topics", rp(constants.AllPermissions.InstructorExpertiseCreate), h.addExpertiseTopic)
	instructors.DELETE("/:id/expertise/topics/:topicRowId", rp(constants.AllPermissions.InstructorExpertiseDelete), h.deleteExpertiseTopicByRow)

	instructors.GET("/:id/expertise/skills", rp(constants.AllPermissions.InstructorExpertiseRead), h.listExpertiseSkills)
	instructors.POST("/:id/expertise/skills", rp(constants.AllPermissions.InstructorExpertiseCreate), h.addExpertiseSkill)
	instructors.DELETE("/:id/expertise/skills/:skillRowId", rp(constants.AllPermissions.InstructorExpertiseDelete), h.deleteExpertiseSkillByRow)

	apps := rg.Group("/instructor-applications")
	apps.GET("", rp(constants.AllPermissions.InstructorApplicationRead), h.listApplications)
	apps.POST("", rp(constants.AllPermissions.InstructorApplicationCreate), h.submitApplication)
	apps.GET("/:id", rp(constants.AllPermissions.InstructorApplicationRead), h.getApplication)
	apps.POST("/:id/approve", rp(constants.AllPermissions.InstructorApplicationApprove), h.approveApplication)
	apps.POST("/:id/reject", rp(constants.AllPermissions.InstructorApplicationReject), h.rejectApplication)
	apps.DELETE("/:id", rp(constants.AllPermissions.InstructorApplicationDelete), h.deleteApplication)

	profiles := rg.Group("/instructor-profiles")
	profiles.GET("", rp(constants.AllPermissions.InstructorProfileRead), h.listProfiles)
	profiles.GET("/me", rp(constants.AllPermissions.InstructorProfileRead), h.getProfileMe)
	profiles.GET("/:id", rp(constants.AllPermissions.InstructorProfileRead), h.getProfileByUser)
	profiles.POST("", rp(constants.AllPermissions.InstructorProfileCreate), h.upsertProfile)
	profiles.PATCH("/:id", rp(constants.AllPermissions.InstructorProfileUpdate), h.upsertProfile)
	profiles.DELETE("/:id", rp(constants.AllPermissions.InstructorProfileDelete), h.deleteProfile)

	tickets := rg.Group("/instructor-tickets")
	tickets.GET("", rp(constants.AllPermissions.InstructorApplicationRead), h.listTickets)
	tickets.POST("", rp(constants.AllPermissions.InstructorApplicationCreate), h.createTicket)
	tickets.POST("/:id/close", rp(constants.AllPermissions.InstructorTicketClose), h.closeTicket)
	tickets.GET("/:id/messages", rp(constants.AllPermissions.InstructorApplicationRead), h.listTicketMessages)
	tickets.POST("/:id/messages", rp(constants.AllPermissions.InstructorApplicationCreate), h.addTicketMessage)

	stubs := rg.Group("/instructor-stubs")
	stubs.GET("/assignments", rp(constants.AllPermissions.InstructorProfileRead), h.comingSoon)
	stubs.GET("/activity-log", rp(constants.AllPermissions.InstructorProfileRead), h.comingSoon)
}
