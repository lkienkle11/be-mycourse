package gormx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	pgDup := &pgconn.PgError{Code: "23505", ConstraintName: "uix_users_slug_active", Message: "duplicate key value violates unique constraint"}
	pgOther := &pgconn.PgError{Code: "23503", ConstraintName: "fk_x", Message: "foreign key violation"}

	cases := []struct {
		name  string
		err   error
		names []string
		want  bool
	}{
		{"nil", nil, nil, false},
		{"pg unique any", pgDup, nil, true},
		{"pg unique scoped match", pgDup, []string{"uix_users_slug_active"}, true},
		{"pg unique scoped other index", pgDup, []string{"uix_users_email"}, false},
		{"pg wrapped", fmt.Errorf("wrap: %w", pgDup), []string{"uix_users_slug_active"}, true},
		{"pg non unique code", pgOther, nil, false},
		{"string duplicate key", errors.New(`ERROR: duplicate key value violates unique constraint "uix_courses_slug_active"`), []string{"uix_courses_slug_active"}, true},
		{"string duplicate key any", errors.New(`ERROR: duplicate key value violates unique constraint "x"`), nil, true},
		{"string sqlite wording", errors.New("UNIQUE constraint failed: users.email"), nil, true},
		{"string scoped mismatch", errors.New(`duplicate key value violates unique constraint "uix_users_email"`), []string{"uix_users_slug_active"}, false},
		{"string unrelated", errors.New("connection reset"), nil, false},
		{"scoped case insensitive", errors.New(`DUPLICATE KEY ... "UIX_COURSES_SLUG_ACTIVE"`), []string{"uix_courses_slug_active"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUniqueViolation(tc.err, tc.names...); got != tc.want {
				t.Fatalf("IsUniqueViolation(%v, %v) = %v, want %v", tc.err, tc.names, got, tc.want)
			}
		})
	}
}
