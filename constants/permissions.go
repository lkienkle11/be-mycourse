package constants

import (
	"fmt"
	"reflect"
	"sort"
)

// Struct tag `perm` is permissions.code in Postgres. Each field's string value is the runtime
// check string (permissions.code_check) — use like TS: constants.CodeProfileRead.CourseRead
// in RequirePermission(...), not a separate .CodeCheck field.

// --- Domain groups (add APIs by adding a string field + perm tag; register nothing else). ---

type codeRbacT struct {
	Manage string `perm:"rbac.manage"`
}

// CodeRbac rbac.* checks (e.g. CodeRbac.Manage).
var CodeRbac = codeRbacT{Manage: "rbac:manage"}

type codeProfileReadT struct {
	CourseRead  string `perm:"profile.course.read"`
	CourseWrite string `perm:"profile.course.write"`
}

// CodeProfileRead profile-related checks (e.g. CodeProfileRead.CourseRead).
var CodeProfileRead = codeProfileReadT{
	CourseRead:  "profile:course:read",
	CourseWrite: "profile:course:write",
}

type codeCourseT struct {
	Read   string `perm:"course.read"`
	Write  string `perm:"course.write"`
	Delete string `perm:"course.delete"`
	Update string `perm:"course.update"`
	Create string `perm:"course.create"`
}

// CodeCourse course.* checks (e.g. CodeCourse.Read).
var CodeCourse = codeCourseT{
	Read:   "course:read",
	Write:  "course:write",
	Delete: "course:delete",
	Update: "course:update",
	Create: "course:create",
}


func allPermissionGroups() []reflect.Value {
	return []reflect.Value{
		reflect.ValueOf(CodeRbac),
		reflect.ValueOf(CodeProfileRead),
		reflect.ValueOf(CodeCourse),
	}
}

func collectEntries() []struct{ Code, CodeCheck string } {
	var out []struct{ Code, CodeCheck string }
	for _, rv := range allPermissionGroups() {
		rt := rv.Type()
		if rt.Kind() != reflect.Struct {
			continue
		}
		for i := 0; i < rt.NumField(); i++ {
			sf := rt.Field(i)
			if sf.PkgPath != "" {
				continue
			}
			code := sf.Tag.Get("perm")
			if code == "" {
				continue
			}
			fv := rv.Field(i)
			if fv.Kind() != reflect.String {
				panic(fmt.Sprintf("constants: permission field %s.%s must be string", rt.Name(), sf.Name))
			}
			out = append(out, struct{ Code, CodeCheck string }{
				Code:      code,
				CodeCheck: fv.String(),
			})
		}
	}
	return out
}

// AllPermissionEntries returns (Code, CodeCheck) sorted by Code for cmd/syncpermissions.
func AllPermissionEntries() []struct{ Code, CodeCheck string } {
	entries := collectEntries()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Code < entries[j].Code })
	return entries
}
