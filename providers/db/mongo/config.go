package mongo

import (
	"strings"
	"time"
)

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN     = "mongodb://db:db@localhost:27017/db"
	defaultOptions = "maxPoolSize=30&minPoolSize=10&maxIdleTimeMS=5000&ssl=false"
	defaultTimeout = 10 * time.Second
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// mongodb://user:password@host:port/dbname.
	// Default: "mongodb://db:db@localhost:27017/db"
	DSN string `envconfig:"optional"`
	// Options is a string with additional options that will be passed
	// to connection. Default: "maxPoolSize=30&minPoolSize=10&maxIdleTimeMS=5000&ssl=false".
	Options string `envconfig:"optional"`
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"optional"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds database connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout time.Duration `envconfig:"optional"`
}

// SetDefault checks connection config. If required field is empty - it will
// be filled with some default value.
// Returns a copy of config.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if cfgCopy.DSN == "" {
		c.DSN = defaultDSN
	}

	if cfgCopy.Options == "" {
		c.Options = defaultOptions
	}

	if cfgCopy.Timeout == 0 {
		c.Timeout = defaultTimeout
	}

	return &cfgCopy
}

// GetDBName return database name from DSN
func (c *Config) GetDBName() string {
	elements := strings.Split(c.DSN, "/")
	return elements[len(elements)-1]
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
