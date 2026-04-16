package constants

import (
	"fmt"
	"reflect"
	"sort"
)

// rolesPermissionT declares which permission_id each role gets. Tags drive cmd/syncrolepermissions
// (role name + perm_id); field values are unused.
type rolesPermissionT struct {
	// sysadmin — full catalog P1–P13
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
	// instructor
	Instructor_ProfileRead          string `role:"instructor" perm_id:"P1"`
	Instructor_CourseRead           string `role:"instructor" perm_id:"P5"`
	Instructor_CourseUpdate         string `role:"instructor" perm_id:"P6"`
	Instructor_CourseDelete         string `role:"instructor" perm_id:"P7"`
	Instructor_CourseInstructorRead string `role:"instructor" perm_id:"P9"`
	Instructor_UserRead             string `role:"instructor" perm_id:"P10"`
	// learner
	Learner_ProfileRead string `role:"learner" perm_id:"P1"`
	Learner_CourseRead  string `role:"learner" perm_id:"P5"`
	Learner_UserRead    string `role:"learner" perm_id:"P10"`
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
