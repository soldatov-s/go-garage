package ch

import (
	//
	"database/sql"
)

// MigrationInCode represents informational struct for database migration
// that was written as Go code.
// When using such migrations you should not use SQL migrations as such
// mix might fuck up everything. This might be changed in future.
type MigrationInCode struct {
	Name string
	Down func(tx *sql.Tx) error
	Up   func(tx *sql.Tx) error
}
