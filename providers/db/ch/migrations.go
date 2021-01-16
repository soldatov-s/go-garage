package ch

import (
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/pressly/goose"
	"github.com/rs/zerolog"
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
	// DatabaseName
	DatabaseName string `envconfig:"optional"`
}

// Check checks migration options. If required field is empty - it will
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

	if cfgCopy.DatabaseName == "" {
		// Default DatabaseName is "default"
		cfgCopy.DatabaseName = "default"
	}

	return &cfgCopy
}

func (c *Enity) setMigrationFlag() {
	c.migratedMutex.Lock()
	c.migrated = true
	// After successful migration we should not attempt to migrate it
	// again if connection to database was re-established.
	c.cfg.Migrate.Action = actionNothing
	c.migratedMutex.Unlock()
}

func (c *Enity) waitConn() {
	for {
		if c.Conn != nil {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
}

func (c *Enity) getCurrentDBVersion() int64 {
	migrationsLog := c.log.With().Str("subsystem", "database migrations").Logger()

	currentDBVersion, gooseerr := goose.GetDBVersion(c.Conn.DB)
	if gooseerr != nil {
		migrationsLog.Fatal().Err(gooseerr).Msg("failed to get database version")
	}

	return currentDBVersion
}

func (c *Enity) migrate(currentDBVersion int64, log *zerolog.Logger) error {
	var err error

	switch {
	case c.cfg.Migrate.Action == actionUp && c.cfg.Migrate.Count == 0:
		log.Info().Msg("applying all unapplied migrations...")

		err = goose.Up(c.Conn.DB, c.cfg.Migrate.Directory)
	case c.cfg.Migrate.Action == actionUp && c.cfg.Migrate.Count != 0:
		newVersion := currentDBVersion + c.cfg.Migrate.Count

		log.Info().Int64("new version", newVersion).Msg("migrating database to specific version")

		err = goose.UpTo(c.Conn.DB, c.cfg.Migrate.Directory, newVersion)
	case c.cfg.Migrate.Action == actionDown && c.cfg.Migrate.Count == 0:
		log.Info().Msg("downgrading database to zero state, you'll need to re-apply migrations!")

		err = goose.DownTo(c.Conn.DB, c.cfg.Migrate.Directory, 0)

		log.Fatal().Msg("database downgraded to zero state. You have to re-apply migrations")
	case c.cfg.Migrate.Action == actionDown && c.cfg.Migrate.Count != 0:
		newVersion := currentDBVersion - c.cfg.Migrate.Count

		log.Info().Int64("new version", newVersion).Msg("downgrading database to specific version")

		err = goose.DownTo(c.Conn.DB, c.cfg.Migrate.Directory, newVersion)
	default:
		log.Fatal().
			Str("action", c.cfg.Migrate.Action).
			Int64("count", c.cfg.Migrate.Count).
			Msg("unsupported set of migration parameters, cannot continue")
	}

	return err
}

func (c *Enity) migrateSchema() error {
	if c.cfg.Migrate.DatabaseName == "" {
		return nil
	}

	_, err := c.Conn.Exec(c.Conn.Rebind("CREATE DATABASE IF NOT EXISTS " + c.cfg.Migrate.DatabaseName))

	return err
}

// Migrates database.
func (c *Enity) Migrate() {
	c.waitConn()
	migrationsLog := c.log.With().Str("subsystem", "database migrations").Logger()

	err := c.migrateSchema()
	if err != nil {
		migrationsLog.Error().Err(err).Msg("failed to execute schema migration")
		c.setMigrationFlag()

		return
	}

	// Ensuring that we're using right database dialect. Without that
	// errors like:
	//
	//   pq: relation "goose_db_version" already exists
	//
	// might appear when that relation actually exists.
	_ = goose.SetDialect("clickhouse")

	currentDBVersion := c.getCurrentDBVersion()
	migrationsLog.Debug().Int64("database version", currentDBVersion).Msg("current database version obtained")

	if err := c.migrate(currentDBVersion, &migrationsLog); err != nil {
		migrationsLog.Fatal().Err(err).Msg("failed to execute migration sequence")
	}

	migrationsLog.Info().Msg("database migrated successfully")
	c.setMigrationFlag()

	// Figure out was migrate-only mode requested?
	if c.cfg.Migrate.Only {
		migrationsLog.Warn().Msg("only database migrations was requested, shutting down")
		os.Exit(0)
	}

	migrationsLog.Info().Msg("migrate-only mode wasn't requested")
}

// MigrationInCode represents informational struct for database migration
// that was written as Go code.
// When using such migrations you should not use SQL migrations as such
// mix might fuck up everything. This might be changed in future.
type MigrationInCode struct {
	Name string
	Down func(tx *sql.Tx) error
	Up   func(tx *sql.Tx) error
}

func (c *Enity) RegisterMigration(migration *MigrationInCode) {
	goose.AddNamedMigration(migration.Name, migration.Up, migration.Down)
}

// SetDatabaseName sets schema for migrations.
func (c *Enity) SetDatabaseName(dbName string) {
	c.cfg.Migrate.DatabaseName = dbName
	// Add dbname as prefix after approved pull request https://github.com/pressly/goose/pull/228
	goose.SetTableName("goose_db_version")
}
