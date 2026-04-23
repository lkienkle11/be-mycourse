package constants

import (
	"fmt"
	"reflect"
	"sort"
)

// rolesPermissionT declares which permission_id each role gets. Tags drive cmd/syncrolepermissions
// (role name + perm_id); field values are unused.
type rolesPermissionT struct {
	// sysadmin — full catalog P1–P25
	Sysadmin_ProfileRead          string `role:"sysadmin" perm_id:"P1"`
	Sysadmin_ProfileUpdate        string `role:"sysadmin" perm_id:"P2"`
	Sysadmin_ProfileDelete        string `role:"sysadmin" perm_id:"P3"`
	Sysadmin_ProfileCreate        string `role:"sysadmin" perm_id:"P4"`
	Sysadmin_CourseRead           string `role:"sysadmin" perm_id:"P5"`
	Sysadmin_CourseUpdate         string `role:"sysadmin" perm_id:"P6"`
	Sysadmin_CourseDelete         string `role:"sysadmin" perm_id:"P7"`
	Sysadmin_CourseCreate         string `role:"sysadmin" perm_id:"P8"`
	Sysadmin_CourseInstructorRead string `role:"sysadmin" perm_id:"P9"`
	Sysadmin_UserRead             string `role:"sysadmin" perm_id:"P10"`
	Sysadmin_UserUpdate           string `role:"sysadmin" perm_id:"P11"`
	Sysadmin_UserDelete           string `role:"sysadmin" perm_id:"P12"`
	Sysadmin_UserCreate           string `role:"sysadmin" perm_id:"P13"`
	Sysadmin_CourseLevelRead      string `role:"sysadmin" perm_id:"P14"`
	Sysadmin_CourseLevelCreate    string `role:"sysadmin" perm_id:"P15"`
	Sysadmin_CourseLevelUpdate    string `role:"sysadmin" perm_id:"P16"`
	Sysadmin_CourseLevelDelete    string `role:"sysadmin" perm_id:"P17"`
	Sysadmin_CategoryRead         string `role:"sysadmin" perm_id:"P18"`
	Sysadmin_CategoryCreate       string `role:"sysadmin" perm_id:"P19"`
	Sysadmin_CategoryUpdate       string `role:"sysadmin" perm_id:"P20"`
	Sysadmin_CategoryDelete       string `role:"sysadmin" perm_id:"P21"`
	Sysadmin_TagRead              string `role:"sysadmin" perm_id:"P22"`
	Sysadmin_TagCreate            string `role:"sysadmin" perm_id:"P23"`
	Sysadmin_TagUpdate            string `role:"sysadmin" perm_id:"P24"`
	Sysadmin_TagDelete            string `role:"sysadmin" perm_id:"P25"`
	// admin — profile + course + user (no course_instructor)
	Admin_ProfileRead   string `role:"admin" perm_id:"P1"`
	Admin_ProfileUpdate string `role:"admin" perm_id:"P2"`
	Admin_ProfileDelete string `role:"admin" perm_id:"P3"`
	Admin_ProfileCreate string `role:"admin" perm_id:"P4"`
	Admin_CourseRead    string `role:"admin" perm_id:"P5"`
	Admin_CourseUpdate  string `role:"admin" perm_id:"P6"`
	Admin_CourseDelete  string `role:"admin" perm_id:"P7"`
	Admin_CourseCreate  string `role:"admin" perm_id:"P8"`
	Admin_UserRead      string `role:"admin" perm_id:"P10"`
	Admin_UserUpdate    string `role:"admin" perm_id:"P11"`
	Admin_UserDelete    string `role:"admin" perm_id:"P12"`
	Admin_UserCreate    string `role:"admin" perm_id:"P13"`
	Admin_CourseLevelRead   string `role:"admin" perm_id:"P14"`
	Admin_CourseLevelCreate string `role:"admin" perm_id:"P15"`
	Admin_CourseLevelUpdate string `role:"admin" perm_id:"P16"`
	Admin_CourseLevelDelete string `role:"admin" perm_id:"P17"`
	Admin_CategoryRead      string `role:"admin" perm_id:"P18"`
	Admin_CategoryCreate    string `role:"admin" perm_id:"P19"`
	Admin_CategoryUpdate    string `role:"admin" perm_id:"P20"`
	Admin_CategoryDelete    string `role:"admin" perm_id:"P21"`
	Admin_TagRead           string `role:"admin" perm_id:"P22"`
	Admin_TagCreate         string `role:"admin" perm_id:"P23"`
	Admin_TagUpdate         string `role:"admin" perm_id:"P24"`
	Admin_TagDelete         string `role:"admin" perm_id:"P25"`
	// instructor
	Instructor_ProfileRead          string `role:"instructor" perm_id:"P1"`
	Instructor_CourseRead           string `role:"instructor" perm_id:"P5"`
	Instructor_CourseUpdate         string `role:"instructor" perm_id:"P6"`
	Instructor_CourseDelete         string `role:"instructor" perm_id:"P7"`
	Instructor_CourseInstructorRead string `role:"instructor" perm_id:"P9"`
	Instructor_UserRead             string `role:"instructor" perm_id:"P10"`
	Instructor_CourseLevelRead      string `role:"instructor" perm_id:"P14"`
	Instructor_CategoryRead         string `role:"instructor" perm_id:"P18"`
	Instructor_TagRead              string `role:"instructor" perm_id:"P22"`
	// learner
	Learner_ProfileRead string `role:"learner" perm_id:"P1"`
	Learner_CourseRead  string `role:"learner" perm_id:"P5"`
	Learner_UserRead    string `role:"learner" perm_id:"P10"`
	Learner_CourseLevelRead string `role:"learner" perm_id:"P14"`
	Learner_CategoryRead    string `role:"learner" perm_id:"P18"`
	Learner_TagRead         string `role:"learner" perm_id:"P22"`
}

// RolePermissions is the catalog instance used with reflect (tags carry role ↔ perm_id).
var RolePermissions = rolesPermissionT{}

// RolePermissionPair is one row for role_permissions rebuild.
type RolePermissionPair struct {
	RoleName string
	PermID   string
}

// AllRolePermissionPairs returns role name + permission_id (P…) from RolePermissions, sorted for stable runs.
func AllRolePermissionPairs() []RolePermissionPair {
	rt := reflect.TypeOf(RolePermissions)
	if rt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("constants: RolePermissions must be struct, got %s", rt.Kind()))
	}
	var out []RolePermissionPair
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		roleName := sf.Tag.Get("role")
		permID := sf.Tag.Get("perm_id")
		if roleName == "" || permID == "" {
			continue
		}
		out = append(out, RolePermissionPair{RoleName: roleName, PermID: permID})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RoleName != out[j].RoleName {
			return out[i].RoleName < out[j].RoleName
		}
		return comparePermissionID(out[i].PermID, out[j].PermID)
	})
	return out
}
