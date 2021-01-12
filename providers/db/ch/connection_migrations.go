package ch

import (
	// stdlib

	"os"
	"strings"
	"time"

	// other
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

type MigrateOptions struct {
	// Action for migration, may be: nothing, up, down
	Action string
	// Count of applied/rollbacked migration
	Count int64
	// Directory is a path to migrate scripts
	Directory string
	// Only migration, exit from service after migration
	Only bool
	// MigrationsType instructs database migration package to use one
	// or another migrations types.
	MigrationsType MigrationsType
}

// Check checks migration options. If required field is empty - it will
// be filled with some default value.
func (mo *MigrateOptions) Check() {
	if mo.MigrationsType == 0 {
		mo.MigrationsType = MigrationTypeSQLFiles
	}

	if mo.Action == "" {
		mo.Action = actionNothing
	}

	mo.Action = strings.ToLower(mo.Action)

	if mo.MigrationsType == MigrationTypeGoCode {
		mo.Directory = "."
	}
}

func (conn *Enity) setMigrationFlag() {
	conn.migratedMutex.Lock()
	conn.migrated = true
	// After successful migration we should not attempt to migrate it
	// again if connection to database was re-established.
	conn.options.MigrateOptions.Action = actionNothing
	conn.migratedMutex.Unlock()
}

func (conn *Enity) waitConn() {
	for {
		if conn.Conn != nil {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
}

func (conn *Enity) getCurrentDBVersion() int64 {
	migrationsLog := conn.log.With().Str("subsystem", "database migrations").Logger()

	currentDBVersion, gooseerr := goose.GetDBVersion(conn.Conn.DB)
	if gooseerr != nil {
		migrationsLog.Fatal().Err(gooseerr).Msg("Failed to get database version")
	}

	return currentDBVersion
}

func (conn *Enity) migrate(currentDBVersion int64, log *zerolog.Logger) error {
	var err error

	switch {
	case conn.options.MigrateOptions.Action == actionUp && conn.options.MigrateOptions.Count == 0:
		log.Info().Msg("Applying all unapplied migrations...")

		err = goose.Up(conn.Conn.DB, conn.options.MigrateOptions.Directory)
	case conn.options.MigrateOptions.Action == actionUp && conn.options.MigrateOptions.Count != 0:
		newVersion := currentDBVersion + conn.options.MigrateOptions.Count

		log.Info().Int64("new version", newVersion).Msg("Migrating database to specific version")

		err = goose.UpTo(conn.Conn.DB, conn.options.MigrateOptions.Directory, newVersion)
	case conn.options.MigrateOptions.Action == actionDown && conn.options.MigrateOptions.Count == 0:
		log.Info().Msg("Downgrading database to zero state, you'll need to re-apply migrations!")

		err = goose.DownTo(conn.Conn.DB, conn.options.MigrateOptions.Directory, 0)

		log.Fatal().Msg("Database downgraded to zero state. You have to re-apply migrations")
	case conn.options.MigrateOptions.Action == actionDown && conn.options.MigrateOptions.Count != 0:
		newVersion := currentDBVersion - conn.options.MigrateOptions.Count

		log.Info().Int64("new version", newVersion).Msg("Downgrading database to specific version")

		err = goose.DownTo(conn.Conn.DB, conn.options.MigrateOptions.Directory, newVersion)
	default:
		log.Fatal().
			Str("action", conn.options.MigrateOptions.Action).
			Int64("count", conn.options.MigrateOptions.Count).
			Msg("Unsupported set of migration parameters, cannot continue")
	}

	return err
}

func (conn *Enity) migrateSchema() error {
	if conn.dbName == "" {
		return nil
	}

	_, err := conn.Conn.Exec(conn.Conn.Rebind("CREATE DATABASE IF NOT EXISTS " + conn.dbName))

	return err
}

// Migrates database.
func (conn *Enity) Migrate() {
	conn.waitConn()
	migrationsLog := conn.log.With().Str("subsystem", "database migrations").Logger()

	err := conn.migrateSchema()
	if err != nil {
		migrationsLog.Error().Err(err).Msg("Failed to execute schema migration")
		conn.setMigrationFlag()

		return
	}

	// Ensuring that we're using right database dialect. Without that
	// errors like:
	//
	//   pq: relation "goose_db_version" already exists
	//
	// might appear when that relation actually exists.
	_ = goose.SetDialect("clickhouse")

	currentDBVersion := conn.getCurrentDBVersion()
	migrationsLog.Debug().Int64("database version", currentDBVersion).Msg("Current database version obtained")

	if err := conn.migrate(currentDBVersion, &migrationsLog); err != nil {
		migrationsLog.Fatal().Err(err).Msg("Failed to execute migration sequence")
	}

	migrationsLog.Info().Msg("Database migrated successfully")
	conn.setMigrationFlag()

	// Figure out was migrate-only mode requested?
	if conn.options.MigrateOptions.Only {
		migrationsLog.Warn().Msg("Only database migrations was requested, shutting down")
		os.Exit(0)
	}

	migrationsLog.Info().Msg("Migrate-only mode wasn't requested")
}

func (conn *Enity) RegisterMigration(migration *MigrationInCode) {
	goose.AddNamedMigration(migration.Name, migration.Up, migration.Down)
}

// SetDatabaseName sets schema for migrations.
func (conn *Enity) SetDatabaseName(dbName string) {
	conn.dbName = dbName
	// Add dbname as prefix after approved pull request https://github.com/pressly/goose/pull/228
	goose.SetTableName("goose_db_version")
}
