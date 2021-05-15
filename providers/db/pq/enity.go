package pq

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/db/migrations"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
	"golang.org/x/sync/errgroup"

	// nolint : a blank import
	_ "github.com/lib/pq"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	Conn *sqlx.DB
	cfg  *Config
	// Queue for bulk operations
	queueWorkerStopped bool
	queue              []*QueueItem
	queueMutex         *sync.Mutex
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
		e.Conn, err = sqlx.Connect("postgres", e.cfg.ComposeDSN())
		if err != nil {
			return errors.Wrap(err, "connect to enity")
		}
		e.GetLogger(ctx).Info().Msg("database connection established")

		// Migrate database.
		m := migrations.NewMigrator(ctx, e.GetFullName(), e.Conn.DB, e.cfg.Migrate)
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

// createMutexConnect initialize connection for mutex
func (e *Enity) createMutexConnect() (*sqlx.DB, error) {
	// Connect to database.
	conn, err := sqlx.Connect("postgres", e.cfg.ComposeDSN())
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

// QueueItemParam is an interface for wrapping passed structure.
type QueueItemParam interface {
	// IsUnique should return false if item is not unique (and therefore
	// should not be processed) and true if item is unique and should
	// be processed. When uniqueness isn't necessary you may return
	// true here.
	IsUnique(conn *sqlx.DB) bool
	// Prepare should prepare items if needed. For example it may parse
	// timestamps from JSON-only fields into database-only ones. It should
	// return true if item is ready to be processed and false if error
	// occurred and item should not be processed. If preparation isn't
	// necessary you may return true here.
	Prepare() bool
}

// QueueItem is a queue element. It will be used in conjunction with
// sqlx's NamedExec() function.
type QueueItem struct {
	// Query is a SQL query with placeholders for NamedExec(). See sqlx's
	// documentation about NamedExec().
	Query string
	// Params should be a structure that describes parameters. Fields
	// should have proper "db" tag to be properly put in database if
	// field name and table columns differs.
	Param QueueItemParam

	// Queue item flags.
	// IsWaitForFlush signals that this item blocking execution flow
	// and true should be sent via WaitForFlush chan.
	// When this flag set query from this item won't be processed.
	// Also all queue items following this item will be re-added
	// in the beginning of queue for later processing.
	IsWaitForFlush bool
	// WaitForFlush is a channel which will receive "true" once item
	// will be processed.
	WaitForFlush chan bool
}

func (e *Enity) AppendToQueue(queueItem *QueueItem) {
	defer e.queueMutex.Unlock()
	e.queueMutex.Lock()
	e.queue = append(e.queue, queueItem)
}

func (e *Enity) recreateQueue() {
	defer e.queueMutex.Unlock()
	e.queueMutex.Lock()
	e.queue = make([]*QueueItem, 0, 10240)
}

// Queue worker goroutine entry point.
func (e *Enity) startQueueWorker(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("starting queue worker")
	e.recreateQueue()

	for {
		select {
		case <-ctx.Done():
			e.GetLogger(ctx).Info().Msg("queue worker stopped")
			e.queueWorkerStopped = true
			return ctx.Err()
		default:
			if err := e.workWithQueue(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("work queue")
			}
		}
		time.Sleep(e.cfg.QueueWorkerTimeout)
	}
}

func (e *Enity) workWithQueue(ctx context.Context) error {
	if e.WeAreShuttingDown || e.Conn == nil {
		return nil
	}

	e.queueMutex.Lock()
	var queriesToProcess []*QueueItem
	queriesToProcess = append(queriesToProcess, e.queue...)
	e.queueMutex.Unlock()
	e.recreateQueue()

	if e.WeAreShuttingDown {
		if len(queriesToProcess) == 0 {
			return nil
		}

		e.GetLogger(ctx).Warn().Int("items in queue", len(queriesToProcess)).Msg("still has items in queue to process, delaying shutdown")
	}

	e.GetLogger(ctx).Debug().Int("items to process", len(queriesToProcess)).Msg("got queries to process")

	if len(queriesToProcess) == 0 {
		e.GetLogger(ctx).Debug().Msg("nothing to process, skipping iteration")
		return nil
	}

	tx, err := e.Conn.Beginx()
	if err != nil {
		e.queueMutex.Lock()
		e.queue = append(e.queue, queriesToProcess...)
		e.queueMutex.Unlock()
		return errors.Wrap(err, "start sql transaction")
	}

	var (
		waitForFlush     bool
		waitForFlushChan chan bool
		reAddRangeStart  int
	)

	for idx, item := range queriesToProcess {
		// If wait-for-flush item here - stop processing queries and
		// flush data.
		if item.IsWaitForFlush {
			waitForFlush = true
			waitForFlushChan = item.WaitForFlush
			reAddRangeStart = idx

			break
		}
		// Every structure should have "Prepare" function, even if
		// it's empty.
		item.Param.Prepare()
		// And IsUnique function that will check if item is unique
		// for us.
		if item.Param.IsUnique(e.Conn) {
			e.GetLogger(ctx).Debug().Msgf("parameters that will be passed to sqlx: %+v", item.Param)

			_, err := tx.NamedExec(item.Query, item.Param)
			if err != nil {
				// Maybe write problematic queries somewhere.
				e.GetLogger(ctx).Error().Err(err).Msg("failed to execute query!")
			}
		} else {
			e.GetLogger(ctx).Warn().Msgf("this item already present in database: %+v", item.Param)
		}
	}

	if err := tx.Commit(); err != nil {
		e.GetLogger(ctx).Err(err).Msg("failed to commit transaction to database, rolling back...")
		errBack := tx.Rollback()
		if errBack != nil {
			return errors.Wrap(err, "rollback failed transaction")
		}
	} else {
		e.GetLogger(ctx).Debug().Msg("sql transaction committed")
	}

	if waitForFlush {
		waitForFlushChan <- true

		e.queueMutex.Lock()
		e.queue = append(queriesToProcess[reAddRangeStart:], e.queue...)
		e.queueMutex.Unlock()
	}

	return nil
}

func (e *Enity) getDBStats() {
	if e.Conn == nil {
		return
	}

	dbStats := e.Conn.DB.Stats()

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_open_connection"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_open_connection",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq open connection right now"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.OpenConnections))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_max_open_connection"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_max_open_connection",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq max open connection"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxOpenConnections))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_in_use"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_in_use",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq connection in use right now"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.InUse))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_wait_duration"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_wait_duration",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq wait duration"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.WaitDuration))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_max_idle_closed"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_max_idle_closed",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq max idle closed"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxIdleClosed))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_max_life_time_closed"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_max_life_time_closed",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq max life time closed"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxLifetimeClosed))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_pq_idle"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_pq_idle",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq idle"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.Idle))
		},
	}
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) GetMetrics(ctx context.Context) (base.MapMetricsOptions, error) {
	e.Metrics[e.GetFullName()+"_status"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_status",
				Help: utils.JoinStrings(" ", e.GetFullName(), "status link to", utils.RedactedDSN(e.cfg.DSN)),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			if e.Conn != nil {
				err := e.Ping(ctx)
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}

	e.getDBStats()
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
