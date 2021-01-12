package rabbitmq

import "time"

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN = "redis://guest:guest@localhost:6379"
)

// RabbitBaseOptions describes struct with base options
type RabbitBaseOptions struct {
	// ExchangeName is a name of rabbitmq exchange
	ExchangeName string `envconfig:"optional"`
	// RoutingKey is a routing key of rabbitmq exchange
	RoutingKey string `envconfig:"optional"`
}

// ConsumerOptions describes struct with options for consumer
type ConsumerOptions struct {
	RabbitBaseOptions
	// RabbitQueue is a name of rabbitmq queue
	RabbitQueue   string `envconfig:"optional"`
	RabbitConsume string `envconfig:"optional"`
}

// PublisherOptions describes struct with options for publisher
type PublisherOptions struct {
	RabbitBaseOptions
}

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// amqp://user:password@host:port.
	// Default: "amqp://guest:guest@rabbitmq:5672"
	DSN string `envconfig:"default=amqp://guest:guest@rabbitmq:5672"`
	// The BackOff policy (exponential back off) will try
	// to establish a connection, and if it fails, will wait some time,
	// then try again and if it fails, wait the same amount of time or longer.
	BackoffPolicy []time.Duration `envconfig:"default=2s;5s;10s;15s;20s;25s"`

	ConsumerOptions  *ConsumerOptions
	PublisherOptions *PublisherOptions
}

// Validate checks connection options. If required field is empty - it will
// be filled with some default value.
func (co *Config) Validate() {
	if co.DSN == "" {
		co.DSN = defaultDSN
	}

	if len(co.BackoffPolicy) == 0 {
		co.BackoffPolicy = []time.Duration{
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
			15 * time.Second,
			20 * time.Second,
			25 * time.Second,
		}
	}
}
