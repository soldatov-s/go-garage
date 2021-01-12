package pq

import (
	"time"

	"github.com/jmoiron/sqlx"
	// nolint : a blank import
	_ "github.com/lib/pq"
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
			conn.log.Info().Msg("Closing database connection...")

			err := conn.Conn.Close()
			if err != nil {
				conn.log.Error().Err(err).Msg("Failed to close database connection")
			}

			conn.Conn = nil
		}

		return true
	}

	// Pinging connection if it's alive (or we think so).
	if conn.Conn != nil {
		err := conn.Conn.Ping()
		if err != nil {
			conn.log.Error().Err(err).Msg("Database connection lost")
			conn.Conn = nil
		}
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if conn.Conn == nil {
		conn.log.Info().Msg("(Re)Establishing connection to PostgreSQL...")

		// Compose DSN.
		dsn := conn.options.DSN
		if conn.options.Options != "" {
			dsn += "?" + conn.options.Options
		}

		// Connect to database.
		dbConn, err := sqlx.Connect("postgres", dsn)
		if err != nil {
			if !conn.options.StartWatcher {
				conn.log.Fatal().
					Err(err).
					Msgf("Failed to connect to PostgreSQL database")
			}

			conn.log.Error().
				Err(err).
				Msgf("Failed to connect to PostgreSQL database, will try to reconnect after %d seconds", conn.options.Timeout)

			return false
		}

		conn.log.Info().Msg("Database connection (re)established")
		conn.Conn = dbConn

		// Migrate database.
		conn.Migrate()

		// Set connection pooling options.
		conn.Conn.SetConnMaxLifetime(conn.options.MaxConnectionLifetime)
		conn.Conn.SetMaxIdleConns(conn.options.MaxIdleConnections)
		conn.Conn.SetMaxOpenConns(conn.options.MaxOpenedConnections)
	}

	return false
}
