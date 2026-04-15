package constants

type roleT struct {
	Sysadmin   string
	Admin      string
	Instructor string
	Learner    string
}

var Role = roleT{
	Sysadmin:   "sysadmin",
	Admin:      "admin",
	Instructor: "instructor",
	Learner:    "learner",
}
