package pq

import (
	"strings"
)

const (
	actionNothing = "nothing"
	actionUp      = "up"
	actionDown    = "down"
)

// MigrationsType represents enumerator for acceptable migration types.
type MigrationsType int

const (
	MigrationTypeSQLFiles MigrationsType = iota
	MigrationTypeGoCode
)

// String returns stringified representation of migrations type.
func (mt MigrationsType) String() string {
	types := [...]string{
		"SQL files",
		"Go code (as functions)",
	}

	return types[mt]
}

type MigrateConfig struct {
	// Action for migration, may be: nothing, up, down
	Action string `envconfig:"optional"`
	// Count of applied/rollbacked migration
	Count int64 `envconfig:"optional"`
	// Directory is a path to migrate scripts
	Directory string `envconfig:"optional"`
	// Only migration, exit from service after migration
	Only bool `envconfig:"optional"`
	// MigrationsType instructs database migration package to use one
	// or another migrations types.
	MigrationsType MigrationsType `envconfig:"optional"`
	// Name of schema in database
	Schema string `envconfig:"optional"`
}

// SetDefault checks migration options. If required field is empty - it will
// be filled with some default value.
func (c *MigrateConfig) SetDefault() *MigrateConfig {
	cfgCopy := *c
	if cfgCopy.MigrationsType == 0 {
		cfgCopy.MigrationsType = MigrationTypeSQLFiles
	}

	if cfgCopy.Action == "" {
		cfgCopy.Action = actionNothing
	}

	cfgCopy.Action = strings.ToLower(c.Action)

	if cfgCopy.MigrationsType == MigrationTypeGoCode {
		cfgCopy.Directory = "."
	}

	if cfgCopy.Schema == "" {
		// Default schema is "production"
		cfgCopy.Schema = "production"
	}

	return &cfgCopy
}
