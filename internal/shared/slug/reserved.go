package slug

// reservedWords are slugs that must never be assigned to a user because they
// collide with routes or system concepts. Users enable the check through the
// availability callback; course and taxonomy slugs are not reserved.
var reservedWords = map[string]struct{}{
	"admin": {}, "administrator": {}, "api": {}, "auth": {}, "catalog": {},
	"course": {}, "courses": {}, "edit": {}, "help": {}, "instructor": {},
	"instructors": {}, "login": {}, "logout": {}, "me": {}, "new": {},
	"null": {}, "register": {}, "root": {}, "settings": {}, "support": {},
	"system": {}, "undefined": {}, "user": {}, "users": {},
}

// IsReserved reports whether s is a reserved word.
func IsReserved(s string) bool {
	_, ok := reservedWords[s]
	return ok
}
