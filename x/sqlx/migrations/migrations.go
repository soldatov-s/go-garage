package migrations

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/pressly/goose"
	"github.com/rs/zerolog"
)

const (
	ActionNothing = "nothing"
	ActionUp      = "up"
	ActionDown    = "down"
)

// Type represents enumerator for acceptable migration types.
type Type int

const (
	TypeSQLFiles Type = iota
	TypeGoCode
)

// String returns stringified representation of migrations type.
func (mt Type) String() string {
	types := [...]string{
		"SQL files",
		"Go code (as functions)",
	}

	return types[mt]
}

type Config struct {
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
	MigrationsType Type `envconfig:"optional"`
	// Name of schema in database
	Schema string `envconfig:"optional"`
}

// SetDefault checks migration options. If required field is empty - it will
// be filled with some default value.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c
	if cfgCopy.MigrationsType == 0 {
		cfgCopy.MigrationsType = TypeSQLFiles
	}

	if cfgCopy.Action == "" {
		cfgCopy.Action = ActionNothing
	}

	cfgCopy.Action = strings.ToLower(c.Action)

	if cfgCopy.MigrationsType == TypeGoCode {
		cfgCopy.Directory = "."
	}

	if cfgCopy.Schema == "" {
		// Default schema is "production"
		cfgCopy.Schema = "production"
	}

	return &cfgCopy
}

type Migrator struct {
	db            *sql.DB
	config        *Config
	migratedMutex sync.Mutex
	migrated      bool
	dialect       string
}

func NewMigrator(dialect string, conn *sql.DB, config *Config) *Migrator {
	return &Migrator{
		db:      conn,
		config:  config,
		dialect: dialect,
	}
}

// Migrates database.
func (m *Migrator) Migrate(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).With().Str("subsystem", "database migrations").Logger()

	err := m.migrateSchema(ctx)
	if err != nil {
		return errors.Wrap(err, "execute schema migration")
	}

	if err := goose.SetDialect(m.dialect); err != nil {
		return errors.Wrap(err, "set dialect")
	}

	currentDBVersion := m.getCurrentDBVersion(ctx)
	logger.Debug().Int64("database version", currentDBVersion).Msg("current database version obtained")

	if err := m.migrate(ctx, currentDBVersion); err != nil {
		return errors.Wrap(err, "execute migration sequence")
	}

	logger.Info().Msg("database migrated successfully")
	m.setMigrationFlag()

	// Figure out was migrate-only mode requested?
	if m.config.Only {
		logger.Warn().Msg("only database migrations was requested, shutting down")
		os.Exit(0)
	}

	logger.Info().Msg("migrate-only mode wasn't requested")
	return nil
}

func (m *Migrator) setMigrationFlag() {
	m.migratedMutex.Lock()
	m.migrated = true
	// After successful migration we should not attempt to migrate it
	// again if connection to database was re-established.
	m.config.Action = ActionNothing
	m.migratedMutex.Unlock()
}

// InCode represents informational struct for database migration
// that was written as Go code.
// When using such migrations you should not use SQL migrations as such
// mix might fuck up everything. This might be changed in future.
type InCode struct {
	Name string
	Down func(tx *sql.Tx) error
	Up   func(tx *sql.Tx) error
}

func RegisterMigration(migration *InCode) {
	goose.AddNamedMigration(migration.Name, migration.Up, migration.Down)
}

// SetSchema sets schema for migrations.
func (m *Migrator) SetSchema(schema string) {
	m.config.Schema = schema
	goose.SetTableName(schema + ".goose_db_version")
}

func (m *Migrator) getCurrentDBVersion(ctx context.Context) int64 {
	logger := zerolog.Ctx(ctx).With().Str("subsystem", "database migrations").Logger()

	currentDBVersion, gooseerr := goose.GetDBVersion(m.db)
	if gooseerr != nil {
		logger.Fatal().Err(gooseerr).Msg("failed to get database version")
	}

	return currentDBVersion
}

func (m *Migrator) migrate(ctx context.Context, currentDBVersion int64) error {
	var err error

	logger := zerolog.Ctx(ctx).With().Str("subsystem", "database migrations").Logger()

	switch {
	case m.config.Action == ActionUp && m.config.Count == 0:
		logger.Info().Msg("applying all unapplied migrations...")

		err = goose.Up(m.db, m.config.Directory)
	case m.config.Action == ActionUp && m.config.Count != 0:
		newVersion := currentDBVersion + m.config.Count

		logger.Info().Int64("new version", newVersion).Msg("migrating database to specific version")

		err = goose.UpTo(m.db, m.config.Directory, newVersion)
	case m.config.Action == ActionDown && m.config.Count == 0:
		logger.Info().Msg("downgrading database to zero state, you'll need to re-apply migrations!")

		err = goose.DownTo(m.db, m.config.Directory, 0)

		logger.Info().Msg("database downgraded to zero state, you have to re-apply migrations")
	case m.config.Action == ActionDown && m.config.Count != 0:
		newVersion := currentDBVersion - m.config.Count

		logger.Info().Int64("new version", newVersion).Msg("downgrading database to specific version")

		err = goose.DownTo(m.db, m.config.Directory, newVersion)
	default:
		logger.Fatal().
			Str("action", m.config.Action).
			Int64("count", m.config.Count).
			Msg("unsupported set of migration parameters, cannot continue")
	}

	return err
}

func (m *Migrator) migrateSchema(ctx context.Context) error {
	if m.config.Schema == "" {
		return nil
	}

	_, err := m.db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+m.config.Schema)

	return err
}
