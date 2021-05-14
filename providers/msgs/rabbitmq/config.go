package rabbitmq

import "time"

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN = "amqp://guest:guest@rabbitmq:5672"
)

// RabbitBaseConfig describes struct with base options
type RabbitBaseConfig struct {
	// ExchangeName is a name of rabbitmq exchange
	ExchangeName string `envconfig:"optional"`
	// RoutingKey is a routing key of rabbitmq exchange
	RoutingKey string `envconfig:"optional"`
}

// ConsumerConfig describes struct with options for consumer
type ConsumerConfig struct {
	RabbitBaseConfig
	// RabbitQueue is a name of rabbitmq queue
	RabbitQueue   string `envconfig:"optional"`
	RabbitConsume string `envconfig:"optional"`
}

// PublisherConfig describes struct with options for publisher
type PublisherConfig struct {
	RabbitBaseConfig
}

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

	Consumer  *ConsumerConfig
	Publisher *PublisherConfig
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
