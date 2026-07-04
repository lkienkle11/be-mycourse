package userpicker

// ActiveUserWhereClause returns SQL conditions for non-disabled, non-banned users (#2).
// Caller must supply named arg @now (Unix seconds).
func ActiveUserWhereClause() string {
	return `
AND u.is_disable = FALSE
AND (u.banned_until IS NULL OR u.banned_until <= @now)`
}

// EligiblePickerWhereClause returns SQL conditions for active, email-confirmed users (#2 + #3).
// Caller must supply named arg @now (Unix seconds).
func EligiblePickerWhereClause() string {
	return ActiveUserWhereClause() + `
AND u.email_confirmed = TRUE`
}
