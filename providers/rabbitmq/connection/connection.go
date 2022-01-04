package rabbitmqcon

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

type Channel struct {
	dsn           string
	backoffPolicy []time.Duration
	conn          *amqp.Connection
	channel       *amqp.Channel
	mu            sync.RWMutex
	isClosed      bool
}

func NewChannel(dsn string, backoffPolicy []time.Duration) (*Channel, error) {
	conn := &Channel{
		dsn:           dsn,
		backoffPolicy: backoffPolicy,
	}

	return conn, nil
}

// OriConn returns original connection to rabbitmq
func (c *Channel) OriConn() *amqp.Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.conn
}

// OriChannel returns original channel to rabbitmq
func (c *Channel) OriChannel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.channel
}

func (c *Channel) Close(_ context.Context) error {
	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "close rabbitMQ connection")
	}
	c.isClosed = true
	return nil
}

func (c *Channel) connect(_ context.Context) error {
	var err error
	if c.conn, err = amqp.Dial(c.dsn); err != nil {
		return errors.Wrap(err, "connect to rabbitMQ")
	}

	if c.channel, err = c.conn.Channel(); err != nil {
		return errors.Wrap(err, "open a channel")
	}

	return nil
}

// Connect auto reconnect to rabbitmq when we lost connection.
func (c *Channel) Connect(ctx context.Context, errorGroup *errgroup.Group) error {
	if !c.isClosed {
		if err := c.connect(ctx); err != nil {
			return errors.Wrap(err, "connect")
		}
	}

	errorGroup.Go(func() error {
		logger := zerolog.Ctx(ctx)
		logger.Info().Msg("starting connection watcher")

		for {
			select {
			case <-ctx.Done():
				logger.Info().Msg("connection watcher stopped")
				return ctx.Err()
			default:
				reason, ok := <-c.channel.NotifyClose(make(chan *amqp.Error))
				if !ok {
					if c.isClosed {
						return nil
					}
					logger.Err(reason).Msg("rabbitMQ channel unexpected closed")

					c.mu.Lock()
					for _, timeout := range c.backoffPolicy {
						if connErr := c.connect(ctx); connErr != nil {
							logger.Err(connErr).Msg("connection failed, trying to reconnect to rabbitMQ")
							time.Sleep(timeout)
							continue
						}
						break
					}
					c.mu.Unlock()
				}
			}
		}
	})

	return nil
}

func (c *Channel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}

func (c *Channel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.channel.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
}

func (c *Channel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.channel.QueueBind(name, key, exchange, noWait, args)
}

func (c *Channel) Consume(
	queue, consumer string,
	autoAck, exclusive, noLocal, noWait bool,
	args amqp.Table) (<-chan amqp.Delivery, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

// nolint:gocritic // pass msg without pointer as in original func in amqp
func (c *Channel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.channel.Publish(exchange, key, mandatory, immediate, msg)
}
