package rabbitmqpub

// RabbitBaseConfig describes struct with base options
type Config struct {
	// ExchangeName is a name of rabbitmq exchange
	ExchangeName string `envconfig:"optional"`
	// RoutingKey is a routing key of rabbitmq exchange
	RoutingKey string `envconfig:"optional"`
}
