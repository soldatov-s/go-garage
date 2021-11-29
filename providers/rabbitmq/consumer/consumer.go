package rabbitmqconsum

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	rabbitmqcon "github.com/soldatov-s/go-garage/providers/rabbitmq/connection"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

// Consumer is a RabbitConsumer
type Consumer struct {
	*base.MetricsStorage
	config *Config
	conn   **rabbitmqcon.Connection
	name   string
}

func NewConsumer(ctx context.Context, name string, config *Config, conn **rabbitmqcon.Connection) (*Consumer, error) {
	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	c := &Consumer{
		MetricsStorage: base.NewMetricsStorage(),
		config:         config,
		name:           name,
		conn:           conn,
	}

	return c, nil
}

func (c *Consumer) connect(_ context.Context) (<-chan amqp.Delivery, error) {
	channel := (*c.conn).Channel()

	if err := channel.ExchangeDeclare(c.config.ExchangeName, "direct", true,
		false, false,
		false, nil); err != nil {
		return nil, errors.Wrap(err, "declare a exchange")
	}

	if _, err := channel.QueueDeclare(
		c.config.RabbitQueue, // name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	); err != nil {
		return nil, errors.Wrap(err, "declare a queue")
	}

	if err := channel.QueueBind(
		c.config.RabbitQueue,  // queue name
		c.config.RoutingKey,   // routing key
		c.config.ExchangeName, // exchange
		false,
		nil,
	); err != nil {
		return nil, errors.Wrap(err, "bind to queue")
	}

	msg, err := channel.Consume(
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

// Subscriber describes struct with options for subscriber
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

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("connection watcher stopped")
			if err := subscriber.Shutdown(ctx); err != nil {
				logger.Err(err).Msg("shutdown handler")
			}
			return ctx.Err()
		default:
			for d := range msg {
				logger.Debug().Msgf("got new event %+v", string(d.Body))
				if errConsume := subscriber.Consume(ctx, d.Body); errConsume != nil {
					logger.Err(errConsume).Msg("consume message")
					continue
				}
				if err := d.Ack(true); err != nil {
					logger.Err(err).Msg("ack")
				}
			}

			errorGroup.Go(func() error {
				return c.subscribe(ctx, errorGroup, subscriber)
			})
			return nil
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
