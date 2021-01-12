package rabbitmq

import (
	"sync"

	"github.com/streadway/amqp"
)

// Consumer is a RabbitConsumer
type Consumer struct {
	ConsumerConfig
	Conn             *amqp.Connection
	Channel          *amqp.Channel
	shutdownConsumer chan bool
	consumerStarted  chan bool
	wg               sync.WaitGroup
}

func NewConsumer(cfg *ConsumerConfig) *Consumer {
	c := &Consumer{}
	c.ConsumerConfig = *cfg
	c.consumerStarted = make(chan bool)
	c.shutdownConsumer = make(chan bool)

	return c
}
