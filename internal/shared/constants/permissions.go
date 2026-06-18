package constants

// allPermissionsT is the canonical catalog: struct tag perm_id = DB permissions.permission_id (PK).
// Each field's string value = permissions.permission_name (JWT / RequirePermission), e.g. AllPermissions.UserRead.
var AllPermissions = struct {
	// Profile
	ProfileRead   string `perm_id:"P1"`
	ProfileUpdate string `perm_id:"P2"`
	ProfileDelete string `perm_id:"P3"`
	ProfileCreate string `perm_id:"P4"`
	// Course
	CourseRead   string `perm_id:"P5"`
	CourseUpdate string `perm_id:"P6"`
	CourseDelete string `perm_id:"P7"`
	CourseCreate string `perm_id:"P8"`
	// Course Instructor
	CourseInstructorRead string `perm_id:"P9"`
	// User
	UserRead   string `perm_id:"P10"`
	UserUpdate string `perm_id:"P11"`
	UserDelete string `perm_id:"P12"`
	UserCreate string `perm_id:"P13"`
	// Course Level
	CourseLevelRead   string `perm_id:"P14"`
	CourseLevelCreate string `perm_id:"P15"`
	CourseLevelUpdate string `perm_id:"P16"`
	CourseLevelDelete string `perm_id:"P17"`
	// Topic (course_topics, formerly category)
	TopicRead   string `perm_id:"P18"`
	TopicCreate string `perm_id:"P19"`
	TopicUpdate string `perm_id:"P20"`
	TopicDelete string `perm_id:"P21"`
	// Course Outcome
	CourseOutcomeRead   string `perm_id:"P30"`
	CourseOutcomeCreate string `perm_id:"P31"`
	CourseOutcomeUpdate string `perm_id:"P32"`
	CourseOutcomeDelete string `perm_id:"P33"`
	// Course Skill
	CourseSkillRead   string `perm_id:"P34"`
	CourseSkillCreate string `perm_id:"P35"`
	CourseSkillUpdate string `perm_id:"P36"`
	CourseSkillDelete string `perm_id:"P37"`
	// Tag
	TagRead   string `perm_id:"P22"`
	TagCreate string `perm_id:"P23"`
	TagUpdate string `perm_id:"P24"`
	TagDelete string `perm_id:"P25"`
	// Media File
	MediaFileRead   string `perm_id:"P26"`
	MediaFileCreate string `perm_id:"P27"`
	MediaFileUpdate string `perm_id:"P28"`
	MediaFileDelete string `perm_id:"P29"`
	// Role modify (scoped admin actions)
	SysadminModify   string `perm_id:"P38"`
	AdminModify      string `perm_id:"P39"`
	InstructorModify string `perm_id:"P40"`
	// Instructor roster
	InstructorRosterRead   string `perm_id:"P41"`
	InstructorRosterCreate string `perm_id:"P42"`
	InstructorRosterDelete string `perm_id:"P43"`
	// Instructor application
	InstructorApplicationRead    string `perm_id:"P44"`
	InstructorApplicationCreate  string `perm_id:"P45"`
	InstructorApplicationUpdate  string `perm_id:"P46"`
	InstructorApplicationDelete  string `perm_id:"P47"`
	InstructorApplicationApprove string `perm_id:"P48"`
	InstructorApplicationReject  string `perm_id:"P49"`
	// Instructor profile
	InstructorProfileRead   string `perm_id:"P50"`
	InstructorProfileCreate string `perm_id:"P51"`
	InstructorProfileUpdate string `perm_id:"P52"`
	InstructorProfileDelete string `perm_id:"P53"`
	// Instructor expertise
	InstructorExpertiseRead   string `perm_id:"P54"`
	InstructorExpertiseCreate string `perm_id:"P55"`
	InstructorExpertiseUpdate string `perm_id:"P56"`
	InstructorExpertiseDelete string `perm_id:"P57"`
	// Instructor ticket
	InstructorTicketClose string `perm_id:"P58"`
	// Course review (admin/sysadmin queue)
	CourseReviewRead    string `perm_id:"P59"`
	CourseReviewApprove string `perm_id:"P60"`
	CourseReviewReject  string `perm_id:"P61"`
	// Course catalog (admin list + trash action)
	CourseCatalogRead  string `perm_id:"P62"`
	CourseCatalogTrash string `perm_id:"P63"`
	// Course trash bin
	CourseTrashRead    string `perm_id:"P64"`
	CourseTrashRestore string `perm_id:"P65"`
	CourseTrashDelete  string `perm_id:"P66"`
}{
	ProfileRead:                  "profile:read",
	ProfileUpdate:                "profile:update",
	ProfileDelete:                "profile:delete",
	ProfileCreate:                "profile:create",
	CourseRead:                   "course:read",
	CourseUpdate:                 "course:update",
	CourseDelete:                 "course:delete",
	CourseCreate:                 "course:create",
	CourseInstructorRead:         "course_instructor:read",
	UserRead:                     "user:read",
	UserUpdate:                   "user:update",
	UserDelete:                   "user:delete",
	UserCreate:                   "user:create",
	CourseLevelRead:              "course_level:read",
	CourseLevelCreate:            "course_level:create",
	CourseLevelUpdate:            "course_level:update",
	CourseLevelDelete:            "course_level:delete",
	TopicRead:                    "topic:read",
	TopicCreate:                  "topic:create",
	TopicUpdate:                  "topic:update",
	TopicDelete:                  "topic:delete",
	CourseOutcomeRead:            "course_outcome:read",
	CourseOutcomeCreate:          "course_outcome:create",
	CourseOutcomeUpdate:          "course_outcome:update",
	CourseOutcomeDelete:          "course_outcome:delete",
	CourseSkillRead:              "course_skill:read",
	CourseSkillCreate:            "course_skill:create",
	CourseSkillUpdate:            "course_skill:update",
	CourseSkillDelete:            "course_skill:delete",
	TagRead:                      "tag:read",
	TagCreate:                    "tag:create",
	TagUpdate:                    "tag:update",
	TagDelete:                    "tag:delete",
	MediaFileRead:                "media_file:read",
	MediaFileCreate:              "media_file:create",
	MediaFileUpdate:              "media_file:update",
	MediaFileDelete:              "media_file:delete",
	SysadminModify:               "sysadmin:modify",
	AdminModify:                  "admin:modify",
	InstructorModify:             "instructor:modify",
	InstructorRosterRead:         "instructor_roster:read",
	InstructorRosterCreate:       "instructor_roster:create",
	InstructorRosterDelete:       "instructor_roster:delete",
	InstructorApplicationRead:    "instructor_application:read",
	InstructorApplicationCreate:  "instructor_application:create",
	InstructorApplicationUpdate:  "instructor_application:update",
	InstructorApplicationDelete:  "instructor_application:delete",
	InstructorApplicationApprove: "instructor_application:approve",
	InstructorApplicationReject:  "instructor_application:reject",
	InstructorProfileRead:        "instructor_profile:read",
	InstructorProfileCreate:      "instructor_profile:create",
	InstructorProfileUpdate:      "instructor_profile:update",
	InstructorProfileDelete:      "instructor_profile:delete",
	InstructorExpertiseRead:      "instructor_expertise:read",
	InstructorExpertiseCreate:    "instructor_expertise:create",
	InstructorExpertiseUpdate:    "instructor_expertise:update",
	InstructorExpertiseDelete:    "instructor_expertise:delete",
	InstructorTicketClose:        "instructor_ticket:close",
	CourseReviewRead:             "course_review:read",
	CourseReviewApprove:          "course_review:approve",
	CourseReviewReject:           "course_review:reject",
	CourseCatalogRead:            "course_catalog:read",
	CourseCatalogTrash:           "course_catalog:trash",
	CourseTrashRead:              "course_trash:read",
	CourseTrashRestore:           "course_trash:restore",
	CourseTrashDelete:            "course_trash:delete",
}
