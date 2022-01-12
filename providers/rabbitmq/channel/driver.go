package rabbitmqchan

import (
	"context"
	"errors"
	"io"

	"github.com/streadway/amqp"
)

// Driver is the Postgres database driver.
type Driver struct {
	conn *amqp.Connection
}

// NewDriver creates driver, conn is a connection to RabbitMQ.
func NewDriver(conn *amqp.Connection) *Driver {
	return &Driver{
		conn: conn,
	}
}

// Connect opens a new channel to the RabbitMQ.
func (d *Driver) Connect(_ context.Context) (io.Closer, error) {
	return d.conn.Channel()
}

func (d *Driver) GetErrBadConn() error {
	return amqp.ErrClosed
}

func (d *Driver) IsErrBadConn(err error) bool {
	return errors.Is(err, amqp.ErrClosed)
}
