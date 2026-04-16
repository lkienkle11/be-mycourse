package constants

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// allPermissionsT is the canonical catalog: struct tag perm_id = DB permissions.permission_id (PK).
// Each field's string value = permissions.permission_name (JWT / RequirePermission), e.g. AllPermissions.UserRead.

type allPermissionsT struct {
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
}

var AllPermissions = allPermissionsT{
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
}

func allPermissionGroups() []reflect.Value {
	return []reflect.Value{reflect.ValueOf(AllPermissions)}
}

// PermissionCatalogEntry is one row from AllPermissions for DB sync.
type PermissionCatalogEntry struct {
	PermissionID   string
	PermissionName string
}

func comparePermissionID(a, b string) bool {
	na, errA := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(a), "P"))
	nb, errB := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(b), "P"))
	if errA != nil || errB != nil {
		return a < b
	}
	return na < nb
}

func collectPermissionCatalogEntries() []PermissionCatalogEntry {
	var out []PermissionCatalogEntry
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
			permID := sf.Tag.Get("perm_id")
			if permID == "" {
				continue
			}
			fv := rv.Field(i)
			if fv.Kind() != reflect.String {
				panic(fmt.Sprintf("constants: permission field %s.%s must be string", rt.Name(), sf.Name))
			}
			out = append(out, PermissionCatalogEntry{
				PermissionID:   permID,
				PermissionName: fv.String(),
			})
		}
	}
	return out
}

// AllPermissionEntries returns catalog rows sorted by perm_id (P1, P2, …) for cmd/syncpermissions and rbacsync.
func AllPermissionEntries() []PermissionCatalogEntry {
	entries := collectPermissionCatalogEntries()
	sort.Slice(entries, func(i, j int) bool {
		return comparePermissionID(entries[i].PermissionID, entries[j].PermissionID)
	})
	return entries
}
