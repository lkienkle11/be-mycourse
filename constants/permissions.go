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
	// Category
	CategoryRead   string `perm_id:"P18"`
	CategoryCreate string `perm_id:"P19"`
	CategoryUpdate string `perm_id:"P20"`
	CategoryDelete string `perm_id:"P21"`
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
}{
	ProfileRead:          "profile:read",
	ProfileUpdate:        "profile:update",
	ProfileDelete:        "profile:delete",
	ProfileCreate:        "profile:create",
	CourseRead:           "course:read",
	CourseUpdate:         "course:update",
	CourseDelete:         "course:delete",
	CourseCreate:         "course:create",
	CourseInstructorRead: "course_instructor:read",
	UserRead:             "user:read",
	UserUpdate:           "user:update",
	UserDelete:           "user:delete",
	UserCreate:           "user:create",
	CourseLevelRead:      "course_level:read",
	CourseLevelCreate:    "course_level:create",
	CourseLevelUpdate:    "course_level:update",
	CourseLevelDelete:    "course_level:delete",
	CategoryRead:         "category:read",
	CategoryCreate:       "category:create",
	CategoryUpdate:       "category:update",
	CategoryDelete:       "category:delete",
	TagRead:              "tag:read",
	TagCreate:            "tag:create",
	TagUpdate:            "tag:update",
	TagDelete:            "tag:delete",
	MediaFileRead:        "media_file:read",
	MediaFileCreate:      "media_file:create",
	MediaFileUpdate:      "media_file:update",
	MediaFileDelete:      "media_file:delete",
}
