package clickhouse

import (
	"strings"
	"time"

	"github.com/soldatov-s/go-garage/x/sqlx/migrations"
)

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN                = "tcp://127.0.0.1:9000"
	defaultOptions            = "database=default&read_timeout=10&write_timeout=20&compress=true&debug=true"
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
	// tcp://host:port.
	// Default: "tcp://127.0.0.1:9000"
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
	// to connection. Default: "database=default&read_timeout=10&write_timeout=20&compress=true&debug=true".
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
	// Migrate struct contains options for migrate
	Migrate *migrations.Config
}

// SetDefault checks connection config. If required field is empty - it will
// be filled with some default value.
// Returns a copy of config.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if c.DSN == "" {
		c.DSN = defaultDSN
	}

	if c.MaxConnectionLifetime == 0 {
		c.MaxConnectionLifetime = defaultMaxConnLifetime
	}

	if c.MaxIdleConnections == 0 {
		c.MaxIdleConnections = defaultMaxIdleConnections
	}

	if c.MaxOpenedConnections == 0 {
		c.MaxOpenedConnections = defaultMaxOpenedConnections
	}

	if c.Options == "" {
		c.Options = defaultOptions
	}

	if c.QueueWorkerTimeout == 0 {
		c.QueueWorkerTimeout = defaultQueueWorkerTimeout
	}

	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}

	cfgCopy.Migrate = c.Migrate.SetDefault()
	return &cfgCopy
}

// ComposeDSN compose DSN
func (c *Config) ComposeDSN() string {
	// Compose DSN.
	dsn := c.DSN
	if c.Options != "" {
		dsn += "?" + c.Options
	}

	return dsn
}

// GetDBName return database name from DSN
func (c *Config) GetDBName() string {
	elements := strings.Split(c.DSN, "/")
	return elements[len(elements)-1]
}
