package rbaccatalog

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
)

func comparePermissionID(a, b string) bool {
	na, errA := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(a), "P"))
	nb, errB := strconv.Atoi(strings.TrimPrefix(strings.ToUpper(b), "P"))
	if errA != nil || errB != nil {
		return a < b
	}
	return na < nb
}

func AllPermissionEntries() []entities.PermissionCatalogEntry {
	rv := reflect.ValueOf(constants.AllPermissions)
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		panic(constants.MsgRBACCatalogStructRequired)
	}
	out := make([]entities.PermissionCatalogEntry, 0, rt.NumField())
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
			panic(fmt.Sprintf("permission field %s.%s must be string", rt.Name(), sf.Name))
		}
		out = append(out, entities.PermissionCatalogEntry{
			PermissionID:   permID,
			PermissionName: fv.String(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return comparePermissionID(out[i].PermissionID, out[j].PermissionID)
	})
	return out
}

func AllRolePermissionPairs() []entities.RolePermissionPair {
	rt := reflect.TypeOf(constants.RolePermissions)
	if rt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("constants.RolePermissions must be struct, got %s", rt.Kind()))
	}
	out := make([]entities.RolePermissionPair, 0, rt.NumField())
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
		out = append(out, entities.RolePermissionPair{RoleName: roleName, PermID: permID})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RoleName != out[j].RoleName {
			return out[i].RoleName < out[j].RoleName
		}
		return comparePermissionID(out[i].PermID, out[j].PermID)
	})
	return out
}
