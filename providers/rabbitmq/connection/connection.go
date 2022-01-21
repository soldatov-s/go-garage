package rabbitmqcon

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	rabbitmqchan "github.com/soldatov-s/go-garage/providers/rabbitmq/channel"
	"github.com/soldatov-s/go-garage/ucpool"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

var ErrClosed = errors.New("conn/channel closed")

type ChannelPoolItemKey struct {
	Queue    string
	Consumer string
	Exchange string
	Key      string
}

type RabbitMQPool struct {
	dsn           string
	backoffPolicy []time.Duration
	connPool      *ucpool.Pool
	isClosed      bool
}

func NewRabbitMQPool(
	ctx context.Context, dsn string, backoffPolicy []time.Duration) (*RabbitMQPool, error) {
	connDriver := NewDriver(dsn)

	conn := &RabbitMQPool{
		dsn:           dsn,
		backoffPolicy: backoffPolicy,
		connPool:      ucpool.OpenPool(ctx, connDriver),
	}

	return conn, nil
}

// Connect connects to rabbitmq and autoreconnect when we lost connection.
func (p *RabbitMQPool) Connect(ctx context.Context, errorGroup *errgroup.Group) (*Connection, error) {
	if p.isClosed {
		return nil, ErrClosed
	}

	c := &Connection{
		rabbitPool: p,
		errorGroup: errorGroup,
	}

	if err := c.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "init conn")
	}

	c.startWatcher(ctx)

	return c, nil
}

func (p *RabbitMQPool) Close(_ context.Context) error {
	p.isClosed = true
	if err := p.connPool.Close(); err != nil {
		return errors.Wrap(err, "close rabbitMQ connection")
	}

	return nil
}

func (p *RabbitMQPool) IsClosed() bool {
	return p.isClosed
}

func (p *RabbitMQPool) Ping(ctx context.Context) error {
	conn, err := p.getConnFromPool(ctx)
	if err != nil {
		return errors.Wrap(err, "get conn from pool")
	}

	if err := conn.Close(); err != nil {
		return errors.Wrap(err, "close conn")
	}
	return nil
}

func (p *RabbitMQPool) Stat() *ucpool.PoolStats {
	return p.connPool.Stats()
}

func (p *RabbitMQPool) SetMaxOpenConns(n int) {
	p.connPool.SetMaxOpenConns(n)
}

func (p *RabbitMQPool) SetMaxIdleConns(n int) {
	p.connPool.SetMaxIdleConns(n)
}

func (p *RabbitMQPool) SetConnMaxLifetime(d time.Duration) {
	p.connPool.SetConnMaxLifetime(d)
}

func (p *RabbitMQPool) SetConnMaxIdleTime(d time.Duration) {
	p.connPool.SetConnMaxIdleTime(d)
}

func (p *RabbitMQPool) getConnFromPool(ctx context.Context) (*ucpool.Conn, error) {
	conn, err := p.connPool.Conn(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get conn from pool")
	}

	return conn, nil
}

type Connection struct {
	errorGroup  *errgroup.Group
	rabbitPool  *RabbitMQPool
	isClosed    bool
	conn        *ucpool.Conn
	mu          sync.RWMutex
	channelPool *ucpool.Pool

	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
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
			if errConnHelper := c.connHelper(ctx, func(rabbitMQConn *amqp.Connection) error {
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
			}); errConnHelper != nil {
				c.mu.Lock()
				for _, timeout := range c.rabbitPool.backoffPolicy {
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
	})
}

func (c *Connection) createChannelPool(ctx context.Context) error {
	if err := c.connHelper(ctx, func(rabbitMQConn *amqp.Connection) error {
		driverChannel := rabbitmqchan.NewDriver(rabbitMQConn)
		c.channelPool = ucpool.OpenPool(ctx, driverChannel)

		c.channelPool.SetMaxOpenConns(c.maxOpenConns)
		c.channelPool.SetMaxIdleConns(c.maxIdleConns)
		c.channelPool.SetConnMaxLifetime(c.connMaxLifetime)
		c.channelPool.SetConnMaxIdleTime(c.connMaxIdleTime)
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
	c.maxOpenConns = n
	c.channelPool.SetMaxOpenConns(n)
}

func (c *Connection) SetMaxIdleChannels(n int) {
	c.maxIdleConns = n
	c.channelPool.SetMaxIdleConns(n)
}

func (c *Connection) SetChannelMaxLifetime(d time.Duration) {
	c.connMaxLifetime = d
	c.channelPool.SetConnMaxLifetime(d)
}

func (c *Connection) SetChannelMaxIdleTime(d time.Duration) {
	c.connMaxIdleTime = d
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

	channel.startWatcher(ctx)

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
				c.mu.Lock()
				for _, timeout := range c.conn.rabbitPool.backoffPolicy {
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
	})
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
