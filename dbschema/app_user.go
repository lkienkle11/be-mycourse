package dbschema

import "mycourse-io-be/constants"

// AppUser is the application end-user account table (not system_privileged_users).
var AppUser appUserNS

type appUserNS struct{}

func (appUserNS) Table() string { return constants.TableAppUsers }
