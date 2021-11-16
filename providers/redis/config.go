package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN     = "redis://rd:rd@localhost:6379"
	defaultTimeout = 10 * time.Second

	// Default pool settings.
	defaultMinIdleConnections   = 10
	defaultMaxOpenedConnections = 30
	defaultMaxConnLifetime      = time.Second * 10
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// redis://user:password@host:port/databaseNumber.
	// Default: "redis://rd:rd@localhost:6379"
	DSN string `envconfig:"optional"`
	// MaxConnectionLifetime specifies maximum connection lifetime
	// for reusage. Default: 10 seconds.
	MaxConnectionLifetime time.Duration `envconfig:"optional"`
	// Minimum number of idle connections which is useful when establishing
	// new connection is slow. Default: 10 connections.
	MinIdleConnections int `envconfig:"optional"`
	// MaxOpenedConnections specify upper limit for opened connections
	// count. Default: 30 connections.
	MaxOpenedConnections int `envconfig:"optional"`
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"optional"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds redis connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout time.Duration `envconfig:"optional"`
}

// SetDefault checks connection options. If required field is empty - it will
// be filled with some default value.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if cfgCopy.DSN == "" {
		cfgCopy.DSN = defaultDSN
	}

	if cfgCopy.MaxConnectionLifetime == 0 {
		cfgCopy.MaxConnectionLifetime = defaultMaxConnLifetime
	}

	if cfgCopy.MinIdleConnections == 0 {
		cfgCopy.MinIdleConnections = defaultMinIdleConnections
	}

	if cfgCopy.MaxOpenedConnections == 0 {
		cfgCopy.MaxOpenedConnections = defaultMaxOpenedConnections
	}

	if cfgCopy.Timeout == 0 {
		cfgCopy.Timeout = defaultTimeout
	}

	return &cfgCopy
}

func (c *Config) Options() (*redis.Options, error) {
	// Connect to database.
	connOptions, err := redis.ParseURL(c.DSN)
	if err != nil {
		return nil, err
	}

	// Set connection pooling options.
	connOptions.MaxConnAge = c.MaxConnectionLifetime
	connOptions.MinIdleConns = c.MinIdleConnections
	connOptions.PoolSize = c.MaxOpenedConnections

	return connOptions, nil
}
