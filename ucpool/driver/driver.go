package driver

import (
	"context"
	"io"
)

// A Connector represents a driver in a fixed configuration
// and can create any number of equivalent Conns for use
// by multiple goroutines.
//
// A Connector can be passed to sql.OpenDB, to allow drivers
// to implement their own sql.DB constructors, or returned by
// DriverContext's OpenConnector method, to allow drivers
// access to context and to avoid repeated parsing of driver
// configuration.
//
// If a Connector implements io.Closer, the sql package's DB.Close
// method will call Close and return error (if any).
type Connector interface {
	// Connect returns a connection to the database.
	// Connect may return a cached connection (one previously
	// closed), but doing so is unnecessary; the sql package
	// maintains a pool of idle connections for efficient re-use.
	//
	// The provided context.Context is for dialing purposes only
	// (see net.DialContext) and should not be stored or used for
	// other purposes. A default timeout should still be used
	// when dialing as a connection pool may call Connect
	// asynchronously to any query.
	//
	// The returned connection is only used by one goroutine at a
	// time.
	Connect(context.Context) (io.Closer, error)
	// GetErrBadConn returns expected err that means an ErrBadConn
	// (bad connection error).
	// This error should be returned by a driver to signal to the sql
	// package that a driver.Conn is in a bad state (such as the server
	// having earlier closed the connection) and the sql package should
	// retry on a new connection.
	//
	// To prevent duplicate operations, ErrBadConn should NOT be returned
	// if there's a possibility that the server might have
	// performed the operation. Even if the server sends back an error,
	// you shouldn't return ErrBadConn.
	GetErrBadConn() error
	// IsErrBadConn returns that checked err is a bad connection error
	IsErrBadConn(err error) bool
}

// SessionResetter may be implemented by Conn to allow drivers to reset the
// session state associated with the connection and to signal a bad connection.
type SessionResetter interface {
	// ResetSession is called prior to executing a query on the connection
	// if the connection has been used before. If the driver returns ErrBadConn
	// the connection is discarded.
	ResetSession(ctx context.Context) error
}

// Validator may be implemented by Conn to allow drivers to
// signal if a connection is valid or if it should be discarded.
//
// If implemented, drivers may return the underlying error from queries,
// even if the connection should be discarded by the connection pool.
type Validator interface {
	// IsValid is called prior to placing the connection into the
	// connection pool. The connection will be discarded if false is returned.
	IsValid() bool
}
