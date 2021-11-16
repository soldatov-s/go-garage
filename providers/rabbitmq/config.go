package rabbitmq

import "time"

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN = "amqp://guest:guest@rabbitmq:5672"
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// amqp://user:password@host:port.
	// Default: "amqp://guest:guest@rabbitmq:5672"
	DSN string `envconfig:"optional"`
	// The BackOff policy (exponential back off) will try
	// to establish a connection, and if it fails, will wait some time,
	// then try again and if it fails, wait the same amount of time or longer.
	BackoffPolicy []time.Duration `envconfig:"optional"`
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"optional"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds database connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout time.Duration `envconfig:"optional"`
}

// Validate checks connection options. If required field is empty - it will
// be filled with some default value.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if cfgCopy.DSN == "" {
		cfgCopy.DSN = defaultDSN
	}

	if len(cfgCopy.BackoffPolicy) == 0 {
		cfgCopy.BackoffPolicy = []time.Duration{
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
			15 * time.Second,
			20 * time.Second,
			25 * time.Second,
		}
	}

	return &cfgCopy
}
