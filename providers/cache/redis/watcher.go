package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/soldatov-s/go-garage/x/rejson"
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
	_, err := c.Conn.Ping(c.Conn.Context()).Result()
	return err
}

// Connection watcher itself.
func (c *Enity) watcher() bool {
	// If we're shutting down - stop connection watcher.
	if c.weAreShuttingDown {
		c.shutdown()
		return true
	}

	if err := c.ping(); err != nil {
		c.log.Error().Err(err).Msg("cache connection lost")
	}

	// If connection is nil - try to establish
	// connection.
	if c.Conn == nil {
		c.log.Info().Msg("establishing connection to cache...")

		// Connect to database.
		connOptions, err := c.cfg.Options()
		if err != nil {
			c.log.Fatal().Err(err).Msgf("failed parse options: %s", err)
		}

		// Set connection pooling options.
		connOptions.MaxConnAge = c.cfg.MaxConnectionLifetime
		connOptions.MinIdleConns = c.cfg.MinIdleConnections
		connOptions.PoolSize = c.cfg.MaxOpenedConnections

		dbConn := redis.NewClient(connOptions)
		if dbConn.Ping(dbConn.Context()).Err() == nil {
			c.log.Info().Msg("cache connection established")
			c.Conn = rejson.ExtendClient(dbConn)
			return false
		}
		if !c.cfg.StartWatcher {
			c.log.Fatal().Err(err).Msgf("failed to connect to cache")
		}

		c.log.Error().Err(err).Msgf("failed to connect to cache, will try to reconnect after %d seconds", c.cfg.Timeout)
	}

	return false
}
