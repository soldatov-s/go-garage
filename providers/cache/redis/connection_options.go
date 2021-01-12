package redis

import (
	"time"
)

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN     = "redis://rd:rd@localhost:6379"
	defaultOptions = "connect_timeout=10"
	defaultTimeout = 10 * time.Second

	// Default pool settings.
	defaultMinIdleConnections   = 10
	defaultMaxOpenedConnections = 30
	defaultMaxConnLifetime      = time.Second * 10
)

// ConnectionOptions represents configuration structure for every
// connection.
type ConnectionOptions struct {
	// DSN is a connection string in form of DSN. Example:
	// redis://user:password@host:port/databaseNumber.
	// Default: "redis://rd:rd@localhost:6379"
	DSN string `envconfig:"optional"`
	// MaxConnectionLifetime specifies maximum connection lifetime
	// for reusage. Default: 10 seconds.
	MaxConnectionLifetime time.Duration `envconfig:"default=10s"`
	// Minimum number of idle connections which is useful when establishing
	// new connection is slow. Default: 10 connections.
	MinIdleConnections int `envconfig:"default=10"`
	// MaxOpenedConnections specify upper limit for opened connections
	// count. Default: 30 connections.
	MaxOpenedConnections int `envconfig:"default=30"`
	// Options is a string with additional options that will be passed
	// to connection. Default: "connect_timeout=10&sslmode=disable".
	Options string `envconfig:"default=connect_timeout=10&sslmode=disable"`
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"default=false"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds redis connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout time.Duration `envconfig:"default=10s"`
	// KeyPrefix is a prefix for eache key in redis
	KeyPrefix string `envconfig:"default=garage_"`
	// ClearTime is a time of live item
	ClearTime time.Duration `envconfig:"default=10s"`
}

// Validate checks connection options. If required field is empty - it will
// be filled with some default value.
func (co *ConnectionOptions) Validate() {
	if co.DSN == "" {
		co.DSN = defaultDSN
	}

	if co.MaxConnectionLifetime == 0 {
		co.MaxConnectionLifetime = defaultMaxConnLifetime
	}

	if co.MinIdleConnections == 0 {
		co.MinIdleConnections = defaultMinIdleConnections
	}

	if co.MaxOpenedConnections == 0 {
		co.MaxOpenedConnections = defaultMaxOpenedConnections
	}

	if co.Options == "" {
		co.Options = defaultOptions
	}

	if co.Timeout == 0 {
		co.Timeout = defaultTimeout
	}
}
