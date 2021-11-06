package migrations

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/pressly/goose"
	"github.com/soldatov-s/go-garage/log"
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
	cfg           *Config
	migratedMutex sync.Mutex
	migrated      bool
	name          string
	dialect       string
}

func NewMigrator(ctx context.Context, name, dialect string, conn *sql.DB, cfg *Config) *Migrator {
	log.FromContext(ctx).
		GetLogger(name+"_"+"migrator", &log.Field{Name: "subsystem", Value: "database migrations"}).
		Info().Msgf("initializing...")

	return &Migrator{
		name:    name,
		db:      conn,
		cfg:     cfg,
		dialect: dialect,
	}
}

// Migrates database.
func (m *Migrator) Migrate(ctx context.Context) error {
	if err := m.waitConn(ctx); err != nil {
		return errors.Wrap(err, "wait connect")
	}
	logger := log.FromContext(ctx).GetLogger(m.name)

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
	if m.cfg.Only {
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
	m.cfg.Action = ActionNothing
	m.migratedMutex.Unlock()
}

func (m *Migrator) waitConn(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if m.db != nil {
				break
			}
			return nil
		}
		time.Sleep(time.Millisecond * 500)
	}
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
	m.cfg.Schema = schema
	goose.SetTableName(schema + ".goose_db_version")
}

func (m *Migrator) getCurrentDBVersion(ctx context.Context) int64 {
	logger := log.FromContext(ctx).GetLogger(m.name)

	currentDBVersion, gooseerr := goose.GetDBVersion(m.db)
	if gooseerr != nil {
		logger.Fatal().Err(gooseerr).Msg("failed to get database version")
	}

	return currentDBVersion
}

func (m *Migrator) migrate(ctx context.Context, currentDBVersion int64) error {
	var err error

	logger := log.FromContext(ctx).GetLogger(m.name)

	switch {
	case m.cfg.Action == ActionUp && m.cfg.Count == 0:
		logger.Info().Msg("applying all unapplied migrations...")

		err = goose.Up(m.db, m.cfg.Directory)
	case m.cfg.Action == ActionUp && m.cfg.Count != 0:
		newVersion := currentDBVersion + m.cfg.Count

		logger.Info().Int64("new version", newVersion).Msg("migrating database to specific version")

		err = goose.UpTo(m.db, m.cfg.Directory, newVersion)
	case m.cfg.Action == ActionDown && m.cfg.Count == 0:
		logger.Info().Msg("downgrading database to zero state, you'll need to re-apply migrations!")

		err = goose.DownTo(m.db, m.cfg.Directory, 0)

		logger.Info().Msg("database downgraded to zero state, you have to re-apply migrations")
	case m.cfg.Action == ActionDown && m.cfg.Count != 0:
		newVersion := currentDBVersion - m.cfg.Count

		logger.Info().Int64("new version", newVersion).Msg("downgrading database to specific version")

		err = goose.DownTo(m.db, m.cfg.Directory, newVersion)
	default:
		logger.Fatal().
			Str("action", m.cfg.Action).
			Int64("count", m.cfg.Count).
			Msg("unsupported set of migration parameters, cannot continue")
	}

	return err
}

func (m *Migrator) migrateSchema(ctx context.Context) error {
	if m.cfg.Schema == "" {
		return nil
	}

	_, err := m.db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+m.cfg.Schema)

	return err
}
