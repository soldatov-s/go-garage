package rabbitmqcon

import (
	"io"

	"github.com/streadway/amqp"
)

// Driver is the Postgres database driver.
type Driver struct{}

// Open opens a new connection to the database. name is a connection string.
// Most users should only use it through database/sql package from the standard
// library.
func (d *Driver) Open(name string) (io.Closer, error) {
	return amqp.Dial(name)
}
