package rabbitmqpool

import "time"

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN = "amqp://guest:guest@rabbitmq:5672"
)

type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// amqp://user:password@host:port.
	// Default: "amqp://guest:guest@rabbitmq:5672"
	DSN string `envconfig:"optional"`
	// The BackOff policy (exponential back off) will try
	// to establish a connection, and if it fails, will wait some time,
	// then try again and if it fails, wait the same amount of time or longer.
	BackoffPolicy []time.Duration `envconfig:"optional"`
	// MaxOpenChannelsPerConn is a maximum number of open channels per connection
	MaxOpenChannelsPerConn int `envconfig:"optional"`
	// MaxIdleChannelsPerConn is a maximum number of channels in the idle channel
	// pool per connection
	MaxIdleChannelsPerConn int `envconfig:"optional"`
	// ChannelMaxLifetime is a maximum amount of time a channel may be reused
	ChannelMaxLifetime time.Duration `envconfig:"optional"`
	// ChannelMaxIdleTime is a maximum amount of time a channel may be idle
	ChannelMaxIdleTime time.Duration `envconfig:"optional"`
	// MaxOpenConns is a maximum number of open connections in the pool
	MaxOpenConns int `envconfig:"optional"`
	// MaxIdleConns is a maximum number of connections in the idle connection pool
	MaxIdleConns int `envconfig:"optional"`
	// ConnMaxLifetime is a maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration `envconfig:"optional"`
	// ConnMaxIdleTime is a maximum amount of time a connection may be idle
	ConnMaxIdleTime time.Duration `envconfig:"optional"`
}

func defaultBackoffPolicy() []time.Duration {
	return []time.Duration{
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		15 * time.Second,
		20 * time.Second,
		25 * time.Second,
	}
}

// Validate checks connection options. If required field is empty - it will
// be filled with some default value.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if cfgCopy.DSN == "" {
		cfgCopy.DSN = defaultDSN
	}

	if len(cfgCopy.BackoffPolicy) == 0 {
		cfgCopy.BackoffPolicy = defaultBackoffPolicy()
	}

	return &cfgCopy
}
