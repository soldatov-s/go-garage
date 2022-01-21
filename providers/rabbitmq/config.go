package rabbitmq

import (
	"time"

	rabbitmqpool "github.com/soldatov-s/go-garage/providers/rabbitmq/pool"
)

const (
	defaultTimeout = 10 * time.Second
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"optional"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds database connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout    time.Duration `envconfig:"optional"`
	PoolConfig *rabbitmqpool.Config
}

// Validate checks connection options. If required field is empty - it will
// be filled with some default value.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	cfgCopy.PoolConfig = cfgCopy.PoolConfig.SetDefault()

	if cfgCopy.Timeout == 0 {
		cfgCopy.Timeout = defaultTimeout
	}

	return &cfgCopy
}
