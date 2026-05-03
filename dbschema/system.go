package dbschema

import "mycourse-io-be/constants"

// System exposes system-level configuration and operator credential tables.
var System systemNS

type systemNS struct{}

func (systemNS) AppConfig() string { return constants.TableSystemAppConfig }

func (systemNS) PrivilegedUsers() string { return constants.TableSystemPrivilegedUsers }
