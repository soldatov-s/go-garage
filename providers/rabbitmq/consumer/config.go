package rabbitmqconsum

// Config describes struct with options for consumer
type Config struct {
	// ExchangeName is a name of rabbitmq exchange
	ExchangeName string `envconfig:"optional"`
	// RoutingKey is a routing key of rabbitmq exchange
	RoutingKey string `envconfig:"optional"`
	// RabbitQueue is a name of rabbitmq queue
	RabbitQueue   string `envconfig:"optional"`
	RabbitConsume string `envconfig:"optional"`
}
