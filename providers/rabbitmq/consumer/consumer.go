package rabbitmqconsum

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	rabbitmqpool "github.com/soldatov-s/go-garage/providers/rabbitmq/pool"
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
	StartWatcher(ctx context.Context, fn rabbitmqpool.WatcherFn) error
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

func (c *Consumer) watcherHandler(ctx context.Context, errorGroup *errgroup.Group, subscriber Subscriber) rabbitmqpool.WatcherFn {
	return func(ctx context.Context) error {
		logger := zerolog.Ctx(ctx)

		logger.Debug().Msg("ExchangeDeclare")
		if err := c.conn.ExchangeDeclare(ctx, c.config.ExchangeName, "direct", true,
			false, false,
			false, nil); err != nil {
			return errors.Wrap(err, "declare a exchange")
		}

		logger.Debug().Msg("QueueDeclare")
		if _, err := c.conn.QueueDeclare(
			ctx,
			c.config.RabbitQueue, // name
			true,                 // durable
			false,                // delete when unused
			false,                // exclusive
			false,                // no-wait
			nil,                  // arguments
		); err != nil {
			return errors.Wrap(err, "declare a queue")
		}

		logger.Debug().Msg("QueueBind")
		if err := c.conn.QueueBind(
			ctx,
			c.config.RabbitQueue,  // queue name
			c.config.RoutingKey,   // routing key
			c.config.ExchangeName, // exchange
			false,
			nil,
		); err != nil {
			return errors.Wrap(err, "bind to queue")
		}

		logger.Debug().Msg("Consume")
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
			return errors.Wrap(err, "consume message")
		}

		errorGroup.Go(func() error {
			logger.Info().Msg("start consumer")
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
						return nil
					}
				}
			}
		})

		return nil
	}
}

// Subscriber describes interface with methods for subscriber
type Subscriber interface {
	Consume(ctx context.Context, data []byte) error
	Shutdown(ctx context.Context) error
}

// Subscribe to channel for receiving message
func (c *Consumer) Subscribe(ctx context.Context, errorGroup *errgroup.Group, subscriber Subscriber) error {
	if errWatcher := c.conn.StartWatcher(ctx, c.watcherHandler(ctx, errorGroup, subscriber)); errWatcher != nil {
		return errors.Wrap(errWatcher, "start watcher")
	}

	return nil
}
