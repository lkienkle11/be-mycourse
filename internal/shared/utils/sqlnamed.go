// Package sqlnamed binds named SQL parameters (:user_id style, like Spring Data @Param + @Query)
// to PostgreSQL positional placeholders ($1, $2, …) using github.com/jmoiron/sqlx.
//
// Table identifiers are not parameters — use dbschema (or fmt.Sprintf on const templates in init),
// then add predicates with :name and pass map[string]interface{} or a struct with `db` tags.
package utils

import "github.com/jmoiron/sqlx"

// Postgres expands :param names in query and returns PostgreSQL-ready SQL plus args in order.
// arg may be map[string]interface{} or a struct with `db:"column"` field tags.
func Postgres(query string, arg interface{}) (string, []interface{}, error) {
	q, args, err := sqlx.Named(query, arg)
	if err != nil {
		return "", nil, err
	}
	q = sqlx.Rebind(sqlx.DOLLAR, q)
	return q, args, nil
}
