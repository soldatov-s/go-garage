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

type ChannelPoolItemKey struct {
	Queue    string
	Consumer string
	Exchange string
	Key      string
}

type Connection struct {
	dsn            string
	backoffPolicy  []time.Duration
	conn           *amqp.Connection
	serviceChannel *amqp.Channel
	mu             sync.RWMutex
	channelPool    map[ChannelPoolItemKey]*amqp.Channel
	channelPoolMu  sync.RWMutex
	isClosed       bool
	errorGroup     *errgroup.Group
	chanCtx        context.Context
}

func NewConnection(dsn string, backoffPolicy []time.Duration) (*Connection, error) {
	conn := &Connection{
		dsn:           dsn,
		backoffPolicy: backoffPolicy,
	}

	return conn, nil
}

// OriConn returns original connection to rabbitmq
func (c *Connection) OriConn() *amqp.Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.conn
}

// Channel returns original channel to rabbitmq
func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	channel, err := c.conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "open a channel")
	}

	return channel, nil
}

func (c *Connection) Close(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isClosed = true

	for _, ch := range c.channelPool {
		if err := ch.Close(); err != nil {
			return errors.Wrap(err, "close rabbitMQ channel")
		}
	}

	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "close rabbitMQ connection")
	}

	return nil
}

func (c *Connection) IsClosed() bool {
	return c.isClosed
}

func (c *Connection) connect(_ context.Context) error {
	var err error
	if c.conn, err = amqp.Dial(c.dsn); err != nil {
		return errors.Wrap(err, "connect to rabbitMQ")
	}

	if c.serviceChannel, err = c.conn.Channel(); err != nil {
		return errors.Wrap(err, "create service rabbitMQ channel")
	}

	c.channelPool = make(map[ChannelPoolItemKey]*amqp.Channel)

	return nil
}

// Connect auto reconnect to rabbitmq when we lost connection.
func (c *Connection) Connect(ctx context.Context, errorGroup *errgroup.Group) error {
	if !c.isClosed {
		if err := c.connect(ctx); err != nil {
			return errors.Wrap(err, "connect")
		}
	}

	c.errorGroup = errorGroup
	c.chanCtx = ctx

	c.errorGroup.Go(func() error {
		logger := zerolog.Ctx(ctx)
		logger.Info().Msg("starting connection watcher")

		for {
			select {
			case <-ctx.Done():
				logger.Info().Msg("connection watcher stopped")
				return ctx.Err()
			default:
				reason, ok := <-c.conn.NotifyClose(make(chan *amqp.Error))
				if !ok {
					if c.isClosed {
						return nil
					}
					logger.Err(reason).Msg("rabbitMQ connection unexpected closed")

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

func (c *Connection) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.serviceChannel.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}

func (c *Connection) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.serviceChannel.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
}

func (c *Connection) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.serviceChannel.QueueBind(name, key, exchange, noWait, args)
}

func (c *Connection) Consume(
	queue, consumer string,
	autoAck, exclusive, noLocal, noWait bool,
	args amqp.Table) (<-chan amqp.Delivery, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ch, err := c.GetChannelFromPool("", "", queue, consumer)
	if err != nil {
		return nil, errors.Wrap(err, "get channel from pool")
	}

	return ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

// nolint:gocritic // pass msg without pointer as in original func in amqp
func (c *Connection) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ch, err := c.GetChannelFromPool(exchange, key, "", "")
	if err != nil {
		return errors.Wrap(err, "get channel from pool")
	}

	return ch.Publish(exchange, key, mandatory, immediate, msg)
}

func (c *Connection) GetChannelFromPool(exchange, key, queue, consumer string) (*amqp.Channel, error) {
	c.channelPoolMu.Lock()
	defer c.channelPoolMu.Unlock()
	var err error
	poolKey := ChannelPoolItemKey{
		Exchange: exchange,
		Key:      key,
		Queue:    queue,
		Consumer: consumer,
	}
	ch, ok := c.channelPool[poolKey]
	if !ok {
		ch, err = c.conn.Channel()
		if err != nil {
			return nil, errors.Wrap(err, "create channel")
		}
		c.channelPool[poolKey] = ch
		c.chanWatcher(poolKey)
	}

	return ch, nil
}

func (c *Connection) chanWatcher(poolKey ChannelPoolItemKey) {
	ch := c.channelPool[poolKey]

	c.errorGroup.Go(func() error {
		logger := zerolog.Ctx(c.chanCtx)
		logger.Info().Msg("starting channel watcher")

		for {
			select {
			case <-c.chanCtx.Done():
				logger.Info().Msg("channel watcher stopped")
				return c.chanCtx.Err()
			default:
				reason, ok := <-ch.NotifyClose(make(chan *amqp.Error))
				if !ok {
					if c.isClosed {
						return nil
					}
					logger.Err(reason).Msg("rabbitMQ channel unexpected closed")
					c.channelPoolMu.Lock()
					delete(c.channelPool, poolKey)
					c.channelPoolMu.Unlock()
					return nil
				}
			}
		}
	})
}
