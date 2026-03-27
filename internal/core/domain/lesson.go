package domain

import "time"

type Lesson struct {
	ID        string
	CourseID  string
	Title     string
	VideoURL  string
	Position  int
	IsPreview bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
