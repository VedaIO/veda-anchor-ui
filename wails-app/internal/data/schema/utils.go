package schema

import (
	"database/sql"
)

// CreateSchema defines and executes the SQL statements to create the database schema.
func CreateSchema(db *sql.DB) error {
	_, err := db.Exec(AppSchema)
	return err
}
