package ch

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/db/migrations"
	"github.com/soldatov-s/go-garage/providers/db/sql"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
	"golang.org/x/sync/errgroup"

	// nolint : a blank import
	_ "github.com/ClickHouse/clickhouse-go"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	Conn *sqlx.DB
	cfg  *Config
	// Queue for bulk operations
	queue              *sql.Queue
	queueWorkerStopped bool
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, collectorName, providerName, name string, cfg interface{}) (*Enity, error) {
	e, err := base.NewEntityWithMetrics(ctx, collectorName, providerName, name)
	if err != nil {
		return nil, errors.Wrap(err, "create base enity")
	}

	// Checking that passed config is OUR.
	config, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityOptions, "expected %q", helper.ObjName(Config{}))
	}

	return &Enity{EntityWithMetrics: e, cfg: config.SetDefault(), queueWorkerStopped: true}, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("shutting down")
	e.WeAreShuttingDown = true

	if e.cfg.StartWatcher {
		for {
			if e.queueWorkerStopped && e.ConnWatcherStopped {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else if err := e.shutdown(ctx); err != nil {
		return errors.Wrapf(err, "shutdown %q", e.GetFullName())
	}

	e.GetLogger(ctx).Info().Msg("shutted down")
	return nil
}

// SetConnPoolLifetime sets connection lifetime.
func (e *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	e.cfg.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if e.Conn != nil {
		e.Conn.SetConnMaxLifetime(connMaxLifetime)
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (e *Enity) SetConnPoolLimits(maxIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	e.cfg.MaxIdleConnections = maxIdleConnections
	e.cfg.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if e.Conn != nil {
		e.Conn.SetMaxIdleConns(maxIdleConnections)
		e.Conn.SetMaxOpenConns(maxOpenedConnections)
	}
}

// SetPoolLimits sets connection pool limits.
func (e *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	e.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	e.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (e *Enity) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	// If connection is nil - try to establish (or reestablish)
	// connection.
	if e.Conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection to database...")
		// Connect to database.
		var err error
		e.Conn, err = sqlx.Connect("clickhouse", e.cfg.ComposeDSN())
		if err != nil {
			return errors.Wrap(err, "connect to enity")
		}
		e.GetLogger(ctx).Info().Msg("database connection established")

		// Migrate database.
		m := migrations.NewMigrator(ctx, e.GetFullName(), "clickhouse", e.Conn.DB, e.cfg.Migrate)
		if err := m.Migrate(ctx); err != nil {
			return errors.Wrap(err, "migrate")
		}

		// Set connection pooling options.
		e.SetConnPoolLifetime(e.cfg.MaxConnectionLifetime)
		e.SetConnPoolLimits(e.cfg.MaxIdleConnections, e.cfg.MaxOpenedConnections)
	}

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.ConnWatcherStopped {
		e.ConnWatcherStopped = false
		errorGroup.Go(func() error {
			return e.startWatcher(ctx)
		})
	}

	// Queue worker will be started only if needed. If it won't be
	// started then queueWorkerStopped flag forced to true.
	if e.cfg.StartQueueWorker && e.queueWorkerStopped {
		e.queueWorkerStopped = false
		errorGroup.Go(func() error {
			return e.startQueueWorker(ctx)
		})
	}

	return nil
}

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("starting connection watcher")

	for {
		select {
		case <-ctx.Done():
			e.GetLogger(ctx).Info().Msg("connection watcher stopped")
			e.ConnWatcherStopped = true
			return ctx.Err()
		default:
			if err := e.Ping(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
			}
		}
		time.Sleep(e.cfg.Timeout)
	}
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.Conn == nil {
		return nil
	}
	e.GetLogger(ctx).Info().Msg("closing connection...")

	err := e.Conn.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	e.Conn = nil

	return nil
}

// Pinging connection if it's alive (or we think so).
func (e *Enity) Ping(ctx context.Context) error {
	if e.Conn == nil {
		return nil
	}

	if err := e.Conn.PingContext(ctx); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

// Queue worker goroutine entry point.
func (e *Enity) startQueueWorker(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("starting queue worker")
	e.queue.RecreateQueue()

	for {
		select {
		case <-ctx.Done():
			e.GetLogger(ctx).Info().Msg("queue worker stopped")
			e.queueWorkerStopped = true
			return ctx.Err()
		default:
			if err := e.queue.ProcessQueue(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("work queue")
			}
		}
		time.Sleep(e.cfg.QueueWorkerTimeout)
	}
}

func (e *Enity) AppendToQueue(queueItem *sql.QueueItem) {
	e.queue.AppendToQueue(queueItem)
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) GetMetrics(ctx context.Context) (base.MapMetricsOptions, error) {
	e.Metrics.AddNewMetricGauge(
		e.GetFullName(),
		"status",
		utils.JoinStrings(" ", "status link to", utils.RedactedDSN(e.cfg.DSN)),
		func() float64 {
			if e.Conn != nil {
				err := e.Ping(ctx)
				if err == nil {
					return 1
				}
			}
			return 0
		},
	)

	sql.GetDBStats(e, e.Conn, e.Metrics)
	return e.Metrics, nil
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	e.ReadyHandlers[strings.ToUpper(e.GetFullName()+"_notfailed")] = func() (bool, string) {
		if e.Conn == nil {
			return false, "not connected"
		}

		if err := e.Ping(ctx); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return e.ReadyHandlers, nil
}
