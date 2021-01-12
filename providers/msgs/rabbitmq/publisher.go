package rabbitmq

import (
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Publisher is a RabbitPublisher
type Publisher struct {
	PublisherConfig
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewPublisher(cfg *PublisherConfig) *Publisher {
	p := &Publisher{}
	p.PublisherConfig = *cfg

	return p
}

func (p *Publisher) Connect(dsn string) error {
	var err error

	p.Conn, err = amqp.Dial(dsn)
	if err != nil {
		return errors.Wrap(err, "publisher can't connect to rabbitMQ: "+dsn)
	}

	p.Channel, err = p.Conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open a channel")
	}

	err = p.Channel.ExchangeDeclare(p.ExchangeName, "direct", true,
		false, false,
		false, nil)
	if err != nil {
		return errors.Wrap(err, "failed to declare a exchange")
	}

	return nil
}

func (p *Publisher) Shutdown() error {
	if p != nil && p.Conn != nil {
		err := p.Channel.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close queue channel")
		}
		err = p.Conn.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close queue connection")
		}

		p.Channel = nil
		p.Conn = nil
	}

	return nil
}
