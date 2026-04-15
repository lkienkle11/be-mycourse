package constants

import (
	"fmt"
	"reflect"
	"sort"
)

// Struct tag `perm` is permissions.code in Postgres. Each field's string value is the runtime
// check string (permissions.action) — use like TS: constants.CodeUser.Read
// in RequirePermission(...), not a separate .Action field.

// --- Domain groups (add APIs by adding a string field + perm tag; register nothing else). ---

type codeUserT struct {
	Read   string `perm:"user.read"`
	Create string `perm:"user.create"`
	Update string `perm:"user.update"`
	Delete string `perm:"user.delete"`
}

// CodeUser user.* checks (e.g. CodeUser.Read).
var CodeUser = codeUserT{
	Read:   "user:read",
	Create: "user:create",
	Update: "user:update",
	Delete: "user:delete",
}

type codeCourseT struct {
	Read   string `perm:"course.read"`
	Create string `perm:"course.create"`
	Update string `perm:"course.update"`
	Delete string `perm:"course.delete"`
}

// CodeCourse course.* checks (e.g. CodeCourse.Read).
var CodeCourse = codeCourseT{
	Read:   "course:read",
	Create: "course:create",
	Update: "course:update",
	Delete: "course:delete",
}

type codeUserAdminT struct {
	Read   string `perm:"user_admin.read"`
	Create string `perm:"user_admin.create"`
	Update string `perm:"user_admin.update"`
	Delete string `perm:"user_admin.delete"`
}

// CodeUserAdmin user_admin.* checks (e.g. CodeUserAdmin.Read).
var CodeUserAdmin = codeUserAdminT{
	Read:   "user_admin:read",
	Create: "user_admin:create",
	Update: "user_admin:update",
	Delete: "user_admin:delete",
}

func allPermissionGroups() []reflect.Value {
	return []reflect.Value{
		reflect.ValueOf(CodeUser),
		reflect.ValueOf(CodeCourse),
		reflect.ValueOf(CodeUserAdmin),
	}
}

func collectEntries() []struct{ Code, Action string } {
	var out []struct{ Code, Action string }
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
			out = append(out, struct{ Code, Action string }{
				Code:   code,
				Action: fv.String(),
			})
		}
	}
	return out
}

// AllPermissionEntries returns (Code, Action) sorted by Code for cmd/syncpermissions.
func AllPermissionEntries() []struct{ Code, Action string } {
	entries := collectEntries()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Code < entries[j].Code })
	return entries
}
