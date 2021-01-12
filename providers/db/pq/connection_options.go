package pq

import (
	"strings"
	"time"
)

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN                = "postgres://db:db@localhost:15432/db"
	defaultOptions            = "connect_timeout=10&sslmode=disable"
	defaultQueueWorkerTimeout = 1 * time.Second
	defaultTimeout            = 10 * time.Second

	// Default pool settings.
	defaultMaxIdleConnections   = 10
	defaultMaxOpenedConnections = 30
	defaultMaxConnLifetime      = time.Second * 10
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// postgres://user:password@host:port/databaseName.
	// Default: "postgres://db:db@localhost:5432/db"
	DSN string `envconfig:"optional"`
	// MaxConnectionLifetime specifies maximum connection lifetime
	// for reusage. Default: 10 seconds.
	MaxConnectionLifetime time.Duration `envconfig:"optional"`
	// MaxIdleConnections specify maximum connections to database that
	// can stay in idle state. Default: 10 connections.
	MaxIdleConnections int `envconfig:"optional"`
	// MaxOpenedConnections specify upper limit for opened connections
	// count. Default: 30 connections.
	MaxOpenedConnections int `envconfig:"optional"`
	// Options is a string with additional options that will be passed
	// to connection. Default: "connect_timeout=10&sslmode=disable".
	Options string `envconfig:"optional"`
	// QueueWorkerTimeout is a timeout in seconds which will be used by
	// queue worker for queue processing. Defaulting to 1. If it'll be
	// set to 0 - it will be reset to 1.
	QueueWorkerTimeout time.Duration `envconfig:"optional"`
	// StartQueueWorker indicates to connection controller that it should
	// also start asynchronous queue worker. This worker can be used for
	// bulking (executing many insert/update/delete requests without
	// big performance penalties).
	StartQueueWorker bool `envconfig:"optional"`
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"optional"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds database connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout time.Duration `envconfig:"optional"`
	// MigrateConfig struct contains options for migrate
	MigrateConfig *MigrateConfig
}

// Validate checks connection options. If required field is empty - it will
// be filled with some default value.
func (co *Config) Validate() {
	if co.DSN == "" {
		co.DSN = defaultDSN
	}

	if co.MaxConnectionLifetime == 0 {
		co.MaxConnectionLifetime = defaultMaxConnLifetime
	}

	if co.MaxIdleConnections == 0 {
		co.MaxIdleConnections = defaultMaxIdleConnections
	}

	if co.MaxOpenedConnections == 0 {
		co.MaxOpenedConnections = defaultMaxOpenedConnections
	}

	if co.Options == "" {
		co.Options = defaultOptions
	}

	if co.QueueWorkerTimeout == 0 {
		co.QueueWorkerTimeout = defaultQueueWorkerTimeout
	}

	if co.Timeout == 0 {
		co.Timeout = defaultTimeout
	}

	co.MigrateConfig.Validate()
}

// GetDBName return database name from DSN
func (co *Config) GetDBName() string {
	elements := strings.Split(co.DSN, "/")
	return elements[len(elements)-1]
}
