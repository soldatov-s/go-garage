package rabbitmqcon

import (
	"context"
	"errors"
	"io"

	"github.com/streadway/amqp"
)

// Driver is the RabbitMQ connection driver.
type Driver struct {
	name string
}

// NewDriver creates driver, name is a connection string.
func NewDriver(name string) *Driver {
	return &Driver{
		name: name,
	}
}

// Connect opens a new connection to the RabbitMQ
func (d *Driver) Connect(_ context.Context) (io.Closer, error) {
	return amqp.Dial(d.name)
}

func (d *Driver) GetErrBadConn() error {
	return amqp.ErrClosed
}

func (d *Driver) IsErrBadConn(err error) bool {
	return errors.Is(err, amqp.ErrClosed)
}
