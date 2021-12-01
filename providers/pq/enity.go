package pq

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
	garageSqlx "github.com/soldatov-s/go-garage/x/sqlx"
	"github.com/soldatov-s/go-garage/x/sqlx/migrations"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"golang.org/x/sync/errgroup"

	// a blank import
	_ "github.com/lib/pq"
)

const ProviderName = "postgres"

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	*base.ReadyCheckStorage
	conn   *sqlx.DB
	config *Config
	// Queue for bulk operations
	queue              *garageSqlx.Queue
	queueWorkerStopped bool
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, config *Config) (*Enity, error) {
	deps := &base.EnityDeps{
		ProviderName: ProviderName,
		Name:         name,
	}
	baseEnity := base.NewEnity(deps)

	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	e := &Enity{
		MetricsStorage:     base.NewMetricsStorage(),
		ReadyCheckStorage:  base.NewReadyCheckStorage(),
		Enity:              baseEnity,
		config:             config.SetDefault(),
		queueWorkerStopped: true,
	}

	if err := e.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	if err := e.buildReadyHandlers(ctx); err != nil {
		return nil, errors.Wrap(err, "build ready handlers")
	}

	return e, nil
}

func (e *Enity) GetConn() *sqlx.DB {
	return e.conn
}

func (e *Enity) GetPConn() **sqlx.DB {
	return &e.conn
}

func (e *Enity) GetConfig() *Config {
	return e.config
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("shutting down")
	e.SetShuttingDown(true)

	if e.config.StartWatcher {
		for {
			if e.queueWorkerStopped && e.IsWatcherStopped() {
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
	e.config.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if e.conn != nil {
		e.conn.SetConnMaxLifetime(connMaxLifetime)
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (e *Enity) SetConnPoolLimits(maxIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	e.config.MaxIdleConnections = maxIdleConnections
	e.config.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if e.conn != nil {
		e.conn.SetMaxIdleConns(maxIdleConnections)
		e.conn.SetMaxOpenConns(maxOpenedConnections)
	}
}

// SetPoolLimits sets connection pool limits.
func (e *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	e.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	e.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (e *Enity) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	logger := e.GetLogger(ctx)

	if e.conn != nil {
		return nil
	}
	logger.Info().Msg("establishing connection...")
	// Connect to database.
	var err error
	e.conn, err = sqlx.Connect(ProviderName, e.config.ComposeDSN())
	if err != nil {
		return errors.Wrap(err, "connect to enity")
	}
	logger.Info().Msg("connection established")
	e.queue = garageSqlx.NewQueue(e.conn)

	// Migrate database.
	m := migrations.NewMigrator(ProviderName, e.conn.DB, e.config.Migrate)
	if err := m.Migrate(ctx); err != nil {
		return errors.Wrap(err, "migrate")
	}

	// Set connection pooling options.
	e.SetConnPoolLifetime(e.config.MaxConnectionLifetime)
	e.SetConnPoolLimits(e.config.MaxIdleConnections, e.config.MaxOpenedConnections)

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.IsWatcherStopped() {
		e.SetWatcher(false)
		errorGroup.Go(func() error {
			return e.startWatcher(ctx)
		})
	}

	// Queue worker will be started only if needed. If it won't be
	// started then queueWorkerStopped flag forced to true.
	if e.config.StartQueueWorker && e.queueWorkerStopped {
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
			e.SetWatcher(true)
			return ctx.Err()
		default:
			if err := e.Ping(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
			}
		}
		time.Sleep(e.config.Timeout)
	}
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}
	e.GetLogger(ctx).Info().Msg("closing connection...")

	if err := e.conn.Close(); err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	e.conn = nil

	return nil
}

// Pinging connection if it's alive (or we think so).
func (e *Enity) Ping(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}

	if err := e.conn.PingContext(ctx); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

// createMutexConnect initialize connection for mutex
func (e *Enity) createMutexConnect() (*sqlx.DB, error) {
	// Connect to database.
	conn, err := sqlx.Connect("postgres", e.config.ComposeDSN())
	if err != nil {
		return nil, errors.Wrap(err, "connect to db")
	}

	// Only one connect in pool for mutex
	conn.SetMaxOpenConns(1)

	return conn, nil
}

// NewMutex create new database mutex
func (e *Enity) NewMutex(checkInterval time.Duration) (*Mutex, error) {
	conn, err := e.createMutexConnect()
	if err != nil {
		return nil, errors.Wrap(err, "create mutex conn")
	}

	return NewMutex(conn, checkInterval)
}

// NewMutexByID create new database mutex with selected id
func (e *Enity) NewMutexByID(lockID int64, checkInterval time.Duration) (*Mutex, error) {
	dbConn, err := e.createMutexConnect()
	if err != nil {
		return nil, errors.Wrap(err, "create mutex conn")
	}

	return NewMutexByID(dbConn, lockID, checkInterval)
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
		time.Sleep(e.config.QueueWorkerTimeout)
	}
}

func (e *Enity) AppendToQueue(queueItem *garageSqlx.QueueItem) {
	e.queue.AppendToQueue(queueItem)
}

// WaitForFlush blocks execution until queue will be empty.
func (e *Enity) WaitForFlush(ctx context.Context) error {
	waitChan := make(chan bool)
	item := &garageSqlx.QueueItem{IsWaitForFlush: true, WaitForFlush: waitChan}
	e.AppendToQueue(item)
	<-waitChan
	e.GetLogger(ctx).Debug().Msg("data flushed to database")

	return nil
}

func (e *Enity) buildMetrics(_ context.Context) error {
	fullName := e.GetFullName()
	redactedDSN, err := stringsx.RedactedDSN(e.config.DSN)
	if err != nil {
		return errors.Wrap(err, "redacted dsn")
	}
	help := stringsx.JoinStrings(" ", "status link to", redactedDSN)
	metricFunc := func(ctx context.Context) (float64, error) {
		if e.conn != nil {
			if errPing := e.Ping(ctx); errPing == nil {
				return 1, nil
			}
		}
		return 0, nil
	}
	if _, errMetrics := e.MetricsStorage.GetMetrics().AddGauge(fullName, "status", help, metricFunc); errMetrics != nil {
		return errors.Wrap(errMetrics, "add gauge metric")
	}

	dbStats, err := garageSqlx.StatsAsMetrics(e)
	if err != nil {
		return errors.Wrap(err, "stats as metrics")
	}

	if err := e.GetMetrics().Append(dbStats); err != nil {
		return errors.Wrap(err, "append db stats")
	}

	return nil
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) buildReadyHandlers(_ context.Context) error {
	checkOptions := &base.CheckOptions{
		Name: strings.ToUpper(e.GetFullName() + "_notfailed"),
		CheckFunc: func(ctx context.Context) error {
			if e.conn == nil {
				return base.ErrNotConnected
			}

			if err := e.Ping(ctx); err != nil {
				return errors.Wrap(err, "ping")
			}

			return nil
		},
	}
	if err := e.ReadyCheckStorage.GetReadyHandlers().Add(checkOptions); err != nil {
		return errors.Wrap(err, "add ready handler")
	}
	return nil
}
