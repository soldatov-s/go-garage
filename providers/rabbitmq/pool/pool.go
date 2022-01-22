package rabbitmqpool

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	rabbitmqchan "github.com/soldatov-s/go-garage/providers/rabbitmq/channel"
	rabbitmqcon "github.com/soldatov-s/go-garage/providers/rabbitmq/connection"
	"github.com/soldatov-s/go-garage/ucpool"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

var ErrClosed = errors.New("conn/channel closed")

type Pool struct {
	config   *Config
	connPool *ucpool.Pool
	isClosed bool
}

func NewPool(ctx context.Context, config *Config) (*Pool, error) {
	connDriver := rabbitmqcon.NewDriver(config.DSN)

	pool := &Pool{
		config:   config,
		connPool: ucpool.OpenPool(ctx, connDriver),
	}

	pool.SetConnMaxIdleTime(pool.config.ConnMaxIdleTime)
	pool.SetConnMaxLifetime(pool.config.ConnMaxLifetime)
	pool.SetMaxIdleConns(pool.config.MaxIdleConns)
	pool.SetMaxOpenConns(pool.config.MaxOpenConns)

	return pool, nil
}

// Connect connects to rabbitmq and autoreconnect when we lost connection.
func (p *Pool) Connect(ctx context.Context, errorGroup *errgroup.Group) (*Connection, error) {
	if p.isClosed {
		return nil, ErrClosed
	}

	c := &Connection{
		rabbitPool:         p,
		errorGroup:         errorGroup,
		maxOpenChannels:    p.config.MaxOpenChannelsPerConn,
		maxIdleChannels:    p.config.MaxIdleChannelsPerConn,
		channelMaxLifetime: p.config.ChannelMaxLifetime,
		channelMaxIdleTime: p.config.ChannelMaxIdleTime,
	}

	if err := c.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "init conn")
	}

	c.startWatcher(ctx)

	return c, nil
}

func (p *Pool) Close(_ context.Context) error {
	p.isClosed = true
	if err := p.connPool.Close(); err != nil {
		return errors.Wrap(err, "close rabbitMQ connection")
	}

	return nil
}

func (p *Pool) IsClosed() bool {
	return p.isClosed
}

func (p *Pool) Ping(ctx context.Context) error {
	conn, err := p.getConnFromPool(ctx)
	if err != nil {
		return errors.Wrap(err, "get conn from pool")
	}

	if err := conn.Close(); err != nil {
		return errors.Wrap(err, "close conn")
	}
	return nil
}

func (p *Pool) Stat() *ucpool.PoolStats {
	return p.connPool.Stats()
}

func (p *Pool) SetMaxOpenConns(n int) {
	p.connPool.SetMaxOpenConns(n)
}

func (p *Pool) SetMaxIdleConns(n int) {
	p.connPool.SetMaxIdleConns(n)
}

func (p *Pool) SetConnMaxLifetime(d time.Duration) {
	p.connPool.SetConnMaxLifetime(d)
}

func (p *Pool) SetConnMaxIdleTime(d time.Duration) {
	p.connPool.SetConnMaxIdleTime(d)
}

func (p *Pool) getConnFromPool(ctx context.Context) (*ucpool.Conn, error) {
	conn, err := p.connPool.Conn(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get conn from pool")
	}

	return conn, nil
}

type Connection struct {
	errorGroup  *errgroup.Group
	rabbitPool  *Pool
	isClosed    bool
	conn        *ucpool.Conn
	mu          sync.RWMutex
	channelPool *ucpool.Pool

	maxOpenChannels    int
	maxIdleChannels    int
	channelMaxLifetime time.Duration
	channelMaxIdleTime time.Duration
}

func (c *Connection) Init(ctx context.Context) error {
	var err error
	c.conn, err = c.rabbitPool.getConnFromPool(ctx)
	if err != nil {
		return errors.Wrap(err, "get conn from pool")
	}

	if err := c.createChannelPool(ctx); err != nil {
		return errors.Wrap(err, "create channel pool")
	}

	return nil
}

func (c *Connection) startWatcher(ctx context.Context) {
	c.errorGroup.Go(func() error {
		logger := zerolog.Ctx(ctx)
		logger.Info().Msg("starting connection watcher")

		for {
			if err := c.notifyClose(ctx); err != nil {
				switch {
				case errors.Is(err, context.Canceled),
					errors.Is(err, context.DeadlineExceeded):
					return err
				default:
					c.mu.Lock()
					// remove the failed conn from pool
					if err := c.conn.Close(); err != nil {
						logger.Err(err).Msg("close conn in pool")
					}
					for _, timeout := range c.rabbitPool.config.BackoffPolicy {
						var connErr error
						if connErr = c.Init(ctx); connErr != nil {
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
}

func (c *Connection) notifyClose(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	if err := c.connHelper(ctx, func(rabbitMQConn *amqp.Connection) error {
		select {
		case <-ctx.Done():
			c.isClosed = true
			logger.Debug().Msg("connection watcher stopped")
			return ctx.Err()
		case reason, ok := <-rabbitMQConn.NotifyClose(make(chan *amqp.Error)):
			if !ok {
				if c.IsClosed() {
					return nil
				}
				return errors.Wrap(reason, "unexpected closed")
			}
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "conn helper")
	}

	return nil
}

func (c *Connection) createChannelPool(ctx context.Context) error {
	if err := c.connHelper(ctx, func(rabbitMQConn *amqp.Connection) error {
		driverChannel := rabbitmqchan.NewDriver(rabbitMQConn)
		c.channelPool = ucpool.OpenPool(ctx, driverChannel)

		c.channelPool.SetMaxOpenConns(c.maxOpenChannels)
		c.channelPool.SetMaxIdleConns(c.maxIdleChannels)
		c.channelPool.SetConnMaxLifetime(c.channelMaxLifetime)
		c.channelPool.SetConnMaxIdleTime(c.channelMaxIdleTime)
		return nil
	}); err != nil {
		return errors.New("open channel pool")
	}

	return nil
}

func (c *Connection) IsClosed() bool {
	return c.isClosed || c.rabbitPool.isClosed
}

func (c *Connection) Close(ctx context.Context) error {
	if c.isClosed {
		return nil
	}

	c.isClosed = true

	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "close connection in pool")
	}

	return nil
}

func (c *Connection) Stat() *ucpool.PoolStats {
	return c.channelPool.Stats()
}

func (c *Connection) SetMaxOpenChannels(n int) {
	c.maxOpenChannels = n
	c.channelPool.SetMaxOpenConns(n)
}

func (c *Connection) SetMaxIdleChannels(n int) {
	c.maxIdleChannels = n
	c.channelPool.SetMaxIdleConns(n)
}

func (c *Connection) SetChannelMaxLifetime(d time.Duration) {
	c.channelMaxLifetime = d
	c.channelPool.SetConnMaxLifetime(d)
}

func (c *Connection) SetChannelMaxIdleTime(d time.Duration) {
	c.channelMaxIdleTime = d
	c.channelPool.SetConnMaxIdleTime(d)
}

func (c *Connection) connHelper(ctx context.Context, fc func(rabbitMQConn *amqp.Connection) error) error {
	driverConn, releaser, err := c.conn.GrabConn(ctx)
	if err != nil {
		return errors.Wrap(err, "GrabConn amqp.Connection")
	}

	// Get conn as *amqp.Connection
	rabbitMQConn, ok := driverConn.GetCi().(*amqp.Connection)
	if !ok {
		return errors.Wrap(err, "typecast to *amqp.Connection")
	}

	driverConn.WithLock(func() {
		err = fc(rabbitMQConn)
	})
	releaser(err)

	return err
}

// Channel return channel object with auto reconnecting when we lost channel.
func (c *Connection) Channel(ctx context.Context) (*Channel, error) {
	if c.isClosed {
		return nil, ErrClosed
	}

	var err error

	channel := &Channel{
		conn: c,
	}
	channel.channel, err = c.channelPool.Conn(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get channel from pool")
	}

	return channel, nil
}

func (c *Connection) ExchangeDeclare(
	ctx context.Context, name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	channel, err := c.Channel(ctx)
	if err != nil {
		return errors.Wrap(err, "create channel")
	}

	if err := channel.ExchangeDeclare(ctx, name, kind, durable, autoDelete, internal, noWait, args); err != nil {
		return errors.Wrap(err, "exchange declare")
	}

	if err := channel.Close(ctx); err != nil {
		return errors.Wrap(err, "return channel to pool")
	}

	return nil
}

func (c *Connection) QueueDeclare(
	ctx context.Context, name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	channel, err := c.Channel(ctx)
	if err != nil {
		return amqp.Queue{}, errors.Wrap(err, "create channel")
	}

	queue, err := channel.QueueDeclare(ctx, name, durable, autoDelete, exclusive, noWait, args)
	if err != nil {
		return amqp.Queue{}, errors.Wrap(err, "queue declare")
	}

	if err := channel.Close(ctx); err != nil {
		return amqp.Queue{}, errors.Wrap(err, "return channel to pool")
	}

	return queue, nil
}

func (c *Connection) QueueBind(ctx context.Context, name, key, exchange string, noWait bool, args amqp.Table) error {
	channel, err := c.Channel(ctx)
	if err != nil {
		return errors.Wrap(err, "create channel")
	}

	if err := channel.QueueBind(ctx, name, key, exchange, noWait, args); err != nil {
		return errors.Wrap(err, "exchange declare")
	}

	if err := channel.Close(ctx); err != nil {
		return errors.Wrap(err, "return channel to pool")
	}

	return nil
}

func (c *Connection) Consume(
	ctx context.Context,
	queue, consumer string,
	autoAck, exclusive, noLocal, noWait bool,
	args amqp.Table) (<-chan amqp.Delivery, error) {
	deliveries := make(chan amqp.Delivery)

	channel, err := c.Channel(ctx)
	if err != nil {
		close(deliveries)
		return deliveries, errors.Wrap(err, "create channel")
	}

	ch, err := channel.Consume(ctx, queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	if err != nil {
		close(deliveries)
		return deliveries, errors.Wrap(err, "exchange declare")
	}

	return ch, nil
}

// nolint:gocritic // pass msg without pointer as in original func in amqp
func (c *Connection) Publish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	channel, err := c.Channel(ctx)
	if err != nil {
		return errors.Wrap(err, "get channel from pool")
	}

	if err := channel.Publish(ctx, exchange, key, mandatory, immediate, msg); err != nil {
		return errors.Wrap(err, "publish")
	}

	if err := channel.Close(ctx); err != nil {
		return errors.Wrap(err, "return channel to pool")
	}

	return nil
}

type Channel struct {
	conn     *Connection
	isClosed bool
	channel  *ucpool.Conn
	mu       sync.RWMutex
}

func (c *Channel) startWatcher(ctx context.Context) {
	c.conn.errorGroup.Go(func() error {
		logger := zerolog.Ctx(ctx)
		logger.Info().Msg("starting channel watcher")

		for {
			if err := c.notifyClose(ctx); err != nil {
				switch {
				case errors.Is(err, context.Canceled),
					errors.Is(err, context.DeadlineExceeded):
					return err
				default:
					c.mu.Lock()
					// remove the failed channel from pool
					if err := c.channel.Close(); err != nil {
						logger.Err(err).Msg("close channel in pool")
					}
					for _, timeout := range c.conn.rabbitPool.config.BackoffPolicy {
						var connErr error
						if c.channel, connErr = c.conn.channelPool.Conn(ctx); connErr != nil {
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
}

func (c *Channel) notifyClose(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	if err := c.channelHelper(ctx, func(rabbitMQChannel *amqp.Channel) error {
		select {
		case <-ctx.Done():
			c.isClosed = true
			logger.Debug().Msg("channel watcher stopped")
			return ctx.Err()
		case reason, ok := <-rabbitMQChannel.NotifyClose(make(chan *amqp.Error)):
			if !ok {
				if c.IsClosed() {
					break
				}
				return errors.Wrap(reason, "unexpected closed")
			}
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "channelHelper")
	}
	return nil
}

func (c *Channel) IsClosed() bool {
	return c.isClosed || c.conn.IsClosed()
}

func (c *Channel) channelHelper(ctx context.Context, fc func(rabbitMQChann *amqp.Channel) error) error {
	driverChannel, releaser, err := c.channel.GrabConn(ctx)
	if err != nil {
		return errors.Wrap(err, "GrabConn amqp.Channel")
	}

	// Get conn as *amqp.Channel
	rabbitMQChann, ok := driverChannel.GetCi().(*amqp.Channel)
	if !ok {
		return errors.Wrap(err, "typecast to *amqp.Channel")
	}

	driverChannel.WithLock(func() {
		err = fc(rabbitMQChann)
	})
	releaser(err)

	return err
}

func (c *Channel) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return nil
	}

	c.isClosed = true

	if err := c.channel.Close(); err != nil {
		return errors.Wrap(err, "close channel in pool")
	}

	return nil
}

func (c *Channel) ExchangeDeclare(
	ctx context.Context, name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsClosed() {
		return ErrClosed
	}

	if err := c.channelHelper(ctx, func(rabbitMQChannel *amqp.Channel) error {
		return rabbitMQChannel.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
	}); err != nil {
		if err != nil {
			return errors.Wrap(err, "channel helper")
		}
	}

	return nil
}

func (c *Channel) QueueDeclare(
	ctx context.Context, name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var queue amqp.Queue
	if c.IsClosed() {
		return queue, ErrClosed
	}

	if err := c.channelHelper(ctx, func(rabbitMQChannel *amqp.Channel) error {
		var errChannel error
		queue, errChannel = rabbitMQChannel.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
		return errChannel
	}); err != nil {
		if err != nil {
			return queue, errors.Wrap(err, "channel helper")
		}
	}

	return queue, nil
}

func (c *Channel) QueueBind(ctx context.Context, name, key, exchange string, noWait bool, args amqp.Table) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsClosed() {
		return ErrClosed
	}

	if err := c.channelHelper(ctx, func(rabbitMQChannel *amqp.Channel) error {
		return rabbitMQChannel.QueueBind(name, key, exchange, noWait, args)
	}); err != nil {
		if err != nil {
			return errors.Wrap(err, "channel helper")
		}
	}

	return nil
}

func (c *Channel) Consume(
	ctx context.Context,
	queue, consumer string,
	autoAck, exclusive, noLocal, noWait bool,
	args amqp.Table) (<-chan amqp.Delivery, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	deliveries := make(chan amqp.Delivery)

	if c.IsClosed() {
		close(deliveries)
		return deliveries, ErrClosed
	}

	var ch <-chan amqp.Delivery

	if err := c.channelHelper(ctx, func(rabbitMQChannel *amqp.Channel) error {
		var errChannel error
		ch, errChannel = rabbitMQChannel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
		return errChannel
	}); err != nil {
		if err != nil {
			close(deliveries)
			return deliveries, errors.Wrap(err, "channel helper")
		}
	}

	c.startWatcher(ctx)

	return ch, nil
}

// nolint:gocritic // pass msg without pointer as in original func in amqp
func (c *Channel) Publish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.IsClosed() {
		return ErrClosed
	}

	if err := c.channelHelper(ctx, func(rabbitMQChannel *amqp.Channel) error {
		return rabbitMQChannel.Publish(exchange, key, mandatory, immediate, msg)
	}); err != nil {
		if err != nil {
			return errors.Wrap(err, "channel helper")
		}
	}

	return nil
}