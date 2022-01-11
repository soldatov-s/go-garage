package rabbitmqchan

import (
	"context"
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
