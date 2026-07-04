package userpicker

import "fmt"

// EligibilitySortTierExpr returns a SQL CASE: 0 = assignment-eligible (#2 + #3), 1 = ineligible.
// Use in ORDER BY … ASC so eligible rows sort first. nowUnix is compared to banned_until.
func EligibilitySortTierExpr(userAlias string, nowUnix int64) string {
	return fmt.Sprintf(`CASE
    WHEN %s.id IS NULL THEN 1
    WHEN %s.is_disable = TRUE THEN 1
    WHEN %s.banned_until IS NOT NULL AND %s.banned_until > %d THEN 1
    WHEN COALESCE(%s.email_confirmed, FALSE) = FALSE THEN 1
    ELSE 0
END`, userAlias, userAlias, userAlias, userAlias, nowUnix, userAlias)
}
