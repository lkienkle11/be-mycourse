package config

type Permission struct {
	Method  string
	Path    string
	Handler string
	Role    string
}

func DefaultPermissions() []Permission {
	return []Permission{
		{
			Method: "GET",
			Path:   "/api/v1/health",
			Role:   "guest",
		},
	}
}
