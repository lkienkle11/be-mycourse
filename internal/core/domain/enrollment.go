package domain

import "time"

type EnrollmentStatus string

const (
	EnrollmentActive    EnrollmentStatus = "active"
	EnrollmentCancelled EnrollmentStatus = "cancelled"
	EnrollmentRefunded  EnrollmentStatus = "refunded"
)

type Enrollment struct {
	ID         string
	StudentID  string
	CourseID   string
	Status     EnrollmentStatus
	EnrolledAt time.Time
}
