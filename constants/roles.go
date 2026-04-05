package constants

type roleT struct {
	Admin   string
	Creator string
	Teacher string
	Learner string
}

var Role = roleT{
	Admin:   "admin",
	Creator: "creator",
	Teacher: "teacher",
	Learner: "learner",
}
