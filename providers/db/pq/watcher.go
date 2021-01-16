package pq

import (
	"time"

	"github.com/jmoiron/sqlx"
	// nolint : a blank import
	_ "github.com/lib/pq"
)

// Connection watcher goroutine entrypoint.
func (c *Enity) startWatcher() {
	c.log.Info().Msg("starting connection watcher")

	ticker := time.NewTicker(c.cfg.Timeout)

	// First start - manually.
	_ = c.watcher()

	// Then - every ticker tick.
	for range ticker.C {
		if c.watcher() {
			break
		}
	}

	ticker.Stop()
	c.log.Info().Msg("connection watcher stopped and connection to database was shutted down")
	c.connWatcherStopped = true
}

func (c *Enity) shutdown() {
	if c.Conn == nil {
		return
	}
	c.log.Info().Msg("closing database connection...")

	err := c.Conn.Close()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to close database connection")
	}

	c.Conn = nil
}

// Pinging connection if it's alive (or we think so).
func (c *Enity) ping() error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Ping()
}

// Connection watcher itself.
func (c *Enity) watcher() bool {
	// If we're shutting down - stop connection watcher.
	if c.weAreShuttingDown {
		c.shutdown()
		return true
	}

	if err := c.ping(); err != nil {
		c.log.Error().Err(err).Msg("database connection lost")
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if c.Conn == nil {
		c.log.Info().Msg("establishing connection to database...")
		// Connect to database.
		dbConn, err := sqlx.Connect("postgres", c.cfg.ComposeDSN())
		if err == nil {
			c.log.Info().Msg("database connection established")
			c.Conn = dbConn

			// Migrate database.
			c.Migrate()

			// Set connection pooling options.
			c.SetConnPoolLifetime(c.cfg.MaxConnectionLifetime)
			c.SetConnPoolLimits(c.cfg.MaxIdleConnections, c.cfg.MaxOpenedConnections)
			return false
		}

		if !c.cfg.StartWatcher {
			c.log.Fatal().Err(err).Msgf("failed to connect to database")
		}

		c.log.Error().Err(err).Msgf("failed to connect to database, reconnect after %d seconds", c.cfg.Timeout)
	}

	return false
}
