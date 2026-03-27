package domain

import "time"

type Course struct {
	ID           string
	InstructorID string
	Title        string
	Description  string
	Price        float64
	ThumbnailURL string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
