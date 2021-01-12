package ch

import (
	//
	"time"

	"github.com/jmoiron/sqlx"
	// ClickHouse driver.
	_ "github.com/ClickHouse/clickhouse-go"
)

// Connection watcher goroutine entrypoint.
func (conn *Enity) startWatcher() {
	conn.log.Info().Msg("starting connection watcher")

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
	conn.log.Info().Msg("connection watcher stopped and connection to database was shutted down")
	conn.connWatcherStopped = true
}

// Connection watcher itself.
func (conn *Enity) watcher() bool {
	// If we're shutting down - stop connection watcher.
	if conn.weAreShuttingDown {
		if conn.Conn != nil {
			conn.log.Info().Msg("closing database connection...")

			err := conn.Conn.Close()
			if err != nil {
				conn.log.Error().Err(err).Msg("failed to close database connection")
			}

			conn.Conn = nil
		}

		return true
	}

	// Pinging connection if it's alive (or we think so).
	if conn.Conn != nil {
		err := conn.Conn.Ping()
		if err != nil {
			conn.log.Error().Err(err).Msg("database connection lost")
			conn.Conn = nil
		}
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if conn.Conn == nil {
		conn.log.Info().Msg("(Re)Establishing connection to ClickHouse...")

		// Compose DSN.
		dsn := conn.options.DSN
		if conn.options.Options != "" {
			dsn += "?" + conn.options.Options
		}

		// Connect to database.
		dbConn, err := sqlx.Connect("clickhouse", dsn)
		if err != nil {
			if !conn.options.StartWatcher {
				conn.log.Fatal().
					Err(err).
					Msgf("failed to connect to ClickHouse database")
			}

			conn.log.Error().
				Err(err).
				Msgf("Failed to connect to ClickHouse database, will try to reconnect after %d seconds", conn.options.Timeout)

			return false
		}

		conn.log.Info().Msg("database connection (re)established")
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
