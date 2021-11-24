package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

type Connection struct {
	dsn           string
	backoffPolicy []time.Duration
	conn          *amqp.Connection
	channel       *amqp.Channel
	mu            sync.Mutex
	isClosed      bool
}

func NewConnection(dsn string, backoffPolicy []time.Duration) (*Connection, error) {
	conn := &Connection{
		dsn:           dsn,
		backoffPolicy: backoffPolicy,
	}

	return conn, nil
}

func (c *Connection) Conn() *amqp.Connection {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn
}

func (c *Connection) Channel() *amqp.Channel {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.channel
}

func (c *Connection) Close(_ context.Context) error {
	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "close rabbitMQ connection")
	}
	c.isClosed = true
	return nil
}

func (c *Connection) connect(_ context.Context) error {
	var err error
	if c.conn, err = amqp.Dial(c.dsn); err != nil {
		return errors.Wrap(err, "connect to rabbitMQ")
	}

	if c.channel, err = c.conn.Channel(); err != nil {
		return errors.Wrap(err, "open a channel")
	}

	return nil
}

// Connection watcher goroutine entrypoint.
func (c *Connection) Connect(ctx context.Context, errorGroup *errgroup.Group) error {
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
