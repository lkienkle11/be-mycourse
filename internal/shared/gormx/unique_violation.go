package gormx

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolationCode is the SQLSTATE PostgreSQL raises for a unique or
// primary-key violation.
const pgUniqueViolationCode = "23505"

// IsUniqueViolation reports whether err is a unique-constraint violation.
// With no constraintNames it matches any unique violation; with names it
// matches only a violation whose constraint/index name (or message) contains
// one of them, so a caller can ignore unrelated unique indexes on the same
// table (for example a slug index versus an email index).
//
// A *pgconn.PgError is inspected by SQLSTATE and constraint name. Other errors
// fall back to a case-insensitive message match on "duplicate key" or
// "unique constraint", which covers drivers that flatten the error to a string
// and the SQLite wording used by tests.
func IsUniqueViolation(err error, constraintNames ...string) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != pgUniqueViolationCode {
			return false
		}
		return matchesAnyName(strings.ToLower(pgErr.ConstraintName+" "+pgErr.Message), constraintNames)
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "duplicate key") && !strings.Contains(msg, "unique constraint") {
		return false
	}
	return matchesAnyName(msg, constraintNames)
}

func matchesAnyName(haystack string, names []string) bool {
	if len(names) == 0 {
		return true
	}
	for _, name := range names {
		if name != "" && strings.Contains(haystack, strings.ToLower(name)) {
			return true
		}
	}
	return false
}
