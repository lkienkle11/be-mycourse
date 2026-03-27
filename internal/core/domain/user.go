package domain

import "time"

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleInstructor Role = "instructor"
	RoleStudent    Role = "student"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
