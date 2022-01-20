package rabbitmqconsum

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen --source=./consumer.go -destination=./consumer_mocks_test.go -package=rabbitmqconsum_test

type Connector interface {
	ExchangeDeclare(ctx context.Context, name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(ctx context.Context, name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(ctx context.Context, name, key, exchange string, noWait bool, args amqp.Table) error
	//nolint:lll // long the function signature
	Consume(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	IsClosed() bool
}

// Consumer is a RabbitConsumer
type Consumer struct {
	*base.MetricsStorage
	config *Config
	conn   Connector
	name   string
}

func NewConsumer(ctx context.Context, config *Config, ch Connector) (*Consumer, error) {
	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	c := &Consumer{
		MetricsStorage: base.NewMetricsStorage(),
		config:         config,
		name:           config.ExchangeName,
		conn:           ch,
	}

	return c, nil
}

func (c *Consumer) connect(ctx context.Context) (<-chan amqp.Delivery, error) {
	if err := c.conn.ExchangeDeclare(ctx, c.config.ExchangeName, "direct", true,
		false, false,
		false, nil); err != nil {
		return nil, errors.Wrap(err, "declare a exchange")
	}

	if _, err := c.conn.QueueDeclare(
		ctx,
		c.config.RabbitQueue, // name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	); err != nil {
		return nil, errors.Wrap(err, "declare a queue")
	}

	if err := c.conn.QueueBind(
		ctx,
		c.config.RabbitQueue,  // queue name
		c.config.RoutingKey,   // routing key
		c.config.ExchangeName, // exchange
		false,
		nil,
	); err != nil {
		return nil, errors.Wrap(err, "bind to queue")
	}

	msg, err := c.conn.Consume(
		ctx,
		c.config.RabbitQueue,   // queue
		c.config.RabbitConsume, // consume
		false,                  // auto-ack
		false,                  // exclusive
		false,                  // no-local
		false,                  // no-wait
		nil,                    // args
	)
	if err != nil {
		return nil, errors.Wrap(err, "consume message")
	}

	return msg, nil
}

// Subscriber describes interface with methods for subscriber
type Subscriber interface {
	Consume(ctx context.Context, data []byte) error
	Shutdown(ctx context.Context) error
}

func (c *Consumer) subscribe(ctx context.Context, errorGroup *errgroup.Group, subscriber Subscriber) error {
	logger := zerolog.Ctx(ctx)
	var msg <-chan amqp.Delivery
	var err error

	for {
		if msg, err = c.connect(ctx); err != nil {
			logger.Err(err).Msg("connect consumer to rabbitMQ")
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	logger.Info().Msg("consumer connected")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("connection watcher stopped")
			if err := subscriber.Shutdown(ctx); err != nil {
				logger.Err(err).Msg("shutdown handler")
			}
			return ctx.Err()
		case d, ok := <-msg:
			if ok {
				logger.Debug().Msgf("got new event %+v", string(d.Body))
				if errConsume := subscriber.Consume(ctx, d.Body); errConsume != nil {
					logger.Err(errConsume).Msg("consume message")
				}
				if err := d.Ack(true); err != nil {
					logger.Err(err).Msg("ack")
				}
			} else {
				if c.conn.IsClosed() {
					return nil
				}

				logger.Info().Msg("try to reconnect consumer")
				errorGroup.Go(func() error {
					return c.subscribe(ctx, errorGroup, subscriber)
				})
				return nil
			}
		}
	}
}

// Subscribe to channel for receiving message
func (c *Consumer) Subscribe(ctx context.Context, errorGroup *errgroup.Group, subscriber Subscriber) error {
	errorGroup.Go(func() error {
		return c.subscribe(ctx, errorGroup, subscriber)
	})

	return nil
}
