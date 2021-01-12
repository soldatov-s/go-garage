package redis

import (
	// stdlib
	"time"

	// other
	"github.com/KromDaniel/rejonson"
	"github.com/go-redis/redis"
)

// Connection watcher goroutine entrypoint.
func (conn *Enity) startWatcher() {
	conn.log.Info().Msg("Starting connection watcher")

	ticker := time.NewTicker(conn.options.Timeout)

	// First start - manually.
	_ = conn.watcher()

	// Then - every ticker tick.
	for range ticker.C {
		if conn.watcher() {
			break
		}
	}

	ticker.Stop()
	conn.log.Info().Msg("Connection watcher stopped and connection to database was shutted down")
	conn.connWatcherStopped = true
}

// Connection watcher itself.
func (conn *Enity) watcher() bool {
	// If we're shutting down - stop connection watcher.
	if conn.weAreShuttingDown {
		if conn.Conn != nil {
			conn.log.Info().Msg("Closing cache connection...")

			err := conn.Conn.Close()
			if err != nil {
				conn.log.Error().Err(err).Msg("Failed to close cache connection")
			}

			conn.Conn = nil
		}

		return true
	}

	// Pinging connection if it's alive (or we think so).
	if conn.Conn != nil {
		_, err := conn.Conn.Ping().Result()
		if err != nil {
			conn.log.Error().Err(err).Msg("Cache connection lost")
			conn.Conn = nil
		}
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if conn.Conn == nil {
		conn.log.Info().Msg("(Re)Establishing connection to Redis...")

		// Connect to database.
		connOptions, err := redis.ParseURL(conn.options.DSN)
		if err != nil {
			conn.log.Error().
				Err(err).
				Msgf("Failed parse DSN to Redis: %s", err)
		}

		// Set connection pooling options.
		connOptions.MaxConnAge = conn.options.MaxConnectionLifetime
		connOptions.MinIdleConns = conn.options.MinIdleConnections
		connOptions.PoolSize = conn.options.MaxOpenedConnections

		dbConn := redis.NewClient(connOptions)
		if dbConn.Ping().Err() != nil {
			if !conn.options.StartWatcher {
				conn.log.Fatal().
					Err(err).
					Msgf("Failed to connect to Redis")
			}

			conn.log.Error().
				Err(err).
				Msgf("Failed to connect to Redis, will try to reconnect after %d seconds", conn.options.Timeout)

			return false
		}

		conn.log.Info().Msg("Cache connection (re)established")
		conn.Conn = rejonson.ExtendClient(dbConn)
	}

	return false
}
