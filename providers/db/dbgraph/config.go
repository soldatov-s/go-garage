package dbgraph

import "time"

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN = "localhost:9080"
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// host:port.
	// Default: "localhost:9080"
	DSN string `envconfig:"optional"`
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

	return &cfgCopy
}
