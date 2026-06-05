package constants

// PostgreSQL relation (table) names used by GORM TableName(), dbschema accessors, and raw SQL.
// Do not import dbschema from this file — avoids import cycles (dbschema imports constants).

// --- RBAC ---

const (
	TableRBACPermissions     = "permissions"
	TableRBACRoles           = "roles"
	TableRBACRolePermissions = "role_permissions"
	TableRBACUserRoles       = "user_roles"
	TableRBACUserPermissions = "user_permissions"
)

// --- Media ---

const (
	TableMediaFiles               = "media_files"
	TableMediaPendingCloudCleanup = "media_pending_cloud_cleanup"
)

// --- Taxonomy ---

const (
	TableTaxonomyTags           = "tags"
	TableTaxonomyCourseTopics   = "course_topics"
	TableTaxonomyCourseOutcomes = "course_outcomes"
	TableTaxonomyCourseSkills   = "course_skills"
	TableTaxonomyCourseLevels   = "course_levels"
)

// --- System (singleton / operators) ---

const (
	TableSystemAppConfig       = "system_app_config"
	TableSystemPrivilegedUsers = "system_privileged_users"
)

// --- Application users (custom users table, BIGINT id) ---

const TableAppUsers = "users"

// --- Instructor ---

const (
	TableInstructorApplications    = "instructor_applications"
	TableInstructorProfiles        = "instructor_profiles"
	TableInstructorExpertiseTopics = "instructor_expertise_topics"
	TableInstructorExpertiseSkills = "instructor_expertise_skills"
	TableInstructorTickets         = "instructor_tickets"
	TableInstructorTicketMessages  = "instructor_ticket_messages"
)

// --- Course ---

const (
	TableCourses                    = "courses"
	TableCourseVersions             = "course_versions"
	TableCourseVersionTags          = "course_version_tags"
	TableCourseVersionSkills        = "course_version_skills"
	TableCourseVersionOutcomes      = "course_version_outcomes"
	TableCourseCollaborators        = "course_collaborators"
	TableCourseSections             = "course_sections"
	TableCourseLessons              = "course_lessons"
	TableCourseSubLessons           = "course_sub_lessons"
	TableCourseSubLessonVideos      = "course_sub_lesson_videos"
	TableCourseSubLessonTexts       = "course_sub_lesson_texts"
	TableCourseSubLessonQuizzes     = "course_sub_lesson_quizzes"
	TableCourseSubLessonQuizOptions = "course_sub_lesson_quiz_options"
	TableCourseEditLeases           = "course_edit_leases"
	TableCourseEnrollments          = "course_enrollments"
	TableCourseProgressItems        = "course_progress_items"
)
