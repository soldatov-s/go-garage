package pq

import (
	//

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
}

// Validate checks migration options. If required field is empty - it will
// be filled with some default value.
func (mo *MigrateConfig) Validate() {
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
	conn.options.MigrateConfig.Action = actionNothing
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
		migrationsLog.Fatal().Err(gooseerr).Msg("failed to get database version")
	}

	return currentDBVersion
}

func (conn *Enity) migrate(currentDBVersion int64, log *zerolog.Logger) error {
	var err error

	switch {
	case conn.options.MigrateConfig.Action == actionUp && conn.options.MigrateConfig.Count == 0:
		log.Info().Msg("applying all unapplied migrations...")

		err = goose.Up(conn.Conn.DB, conn.options.MigrateConfig.Directory)
	case conn.options.MigrateConfig.Action == actionUp && conn.options.MigrateConfig.Count != 0:
		newVersion := currentDBVersion + conn.options.MigrateConfig.Count

		log.Info().Int64("new version", newVersion).Msg("migrating database to specific version")

		err = goose.UpTo(conn.Conn.DB, conn.options.MigrateConfig.Directory, newVersion)
	case conn.options.MigrateConfig.Action == actionDown && conn.options.MigrateConfig.Count == 0:
		log.Info().Msg("downgrading database to zero state, you'll need to re-apply migrations!")

		err = goose.DownTo(conn.Conn.DB, conn.options.MigrateConfig.Directory, 0)

		log.Fatal().Msg("database downgraded to zero state, you have to re-apply migrations")
	case conn.options.MigrateConfig.Action == actionDown && conn.options.MigrateConfig.Count != 0:
		newVersion := currentDBVersion - conn.options.MigrateConfig.Count

		log.Info().Int64("new version", newVersion).Msg("downgrading database to specific version")

		err = goose.DownTo(conn.Conn.DB, conn.options.MigrateConfig.Directory, newVersion)
	default:
		log.Fatal().
			Str("action", conn.options.MigrateConfig.Action).
			Int64("count", conn.options.MigrateConfig.Count).
			Msg("unsupported set of migration parameters, cannot continue")
	}

	return err
}

func (conn *Enity) migrateSchema() error {
	if conn.schemaName == "" {
		return nil
	}

	_, err := conn.Conn.Exec(conn.Conn.Rebind("CREATE SCHEMA IF NOT EXISTS " + conn.schemaName))

	return err
}

// Migrates database.
func (conn *Enity) Migrate() {
	conn.waitConn()
	migrationsLog := conn.log.With().Str("subsystem", "database migrations").Logger()

	err := conn.migrateSchema()
	if err != nil {
		migrationsLog.Error().Err(err).Msg("failed to execute schema migration")
		conn.setMigrationFlag()

		return
	}

	// Ensuring that we're using right database dialect. Without that
	// errors like:
	//
	//   pq: relation "goose_db_version" already exists
	//
	// might appear when that relation actually exists.
	_ = goose.SetDialect("postgres")

	currentDBVersion := conn.getCurrentDBVersion()
	migrationsLog.Debug().Int64("database version", currentDBVersion).Msg("current database version obtained")

	if err := conn.migrate(currentDBVersion, &migrationsLog); err != nil {
		migrationsLog.Fatal().Err(err).Msg("failed to execute migration sequence")
	}

	migrationsLog.Info().Msg("database migrated successfully")
	conn.setMigrationFlag()

	// Figure out was migrate-only mode requested?
	if conn.options.MigrateConfig.Only {
		migrationsLog.Warn().Msg("only database migrations was requested, shutting down")
		os.Exit(0)
	}

	migrationsLog.Info().Msg("migrate-only mode wasn't requested")
}

func (conn *Enity) RegisterMigration(migration *MigrationInCode) {
	goose.AddNamedMigration(migration.Name, migration.Up, migration.Down)
}

// SetSchemaName sets schema for migrations.
func (conn *Enity) SetSchemaName(schemaName string) {
	conn.schemaName = schemaName
	goose.SetTableName(schemaName + ".goose_db_version")
}
