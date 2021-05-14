package ch

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/pressly/goose"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"

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
	// For migration
	migratedMutex sync.Mutex
	migrated      bool
	// Queue for bulk operations
	queue              []*QueueItem
	queueMutex         *sync.Mutex
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
func (e *Enity) Start(ctx context.Context) error {
	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.ConnWatcherStopped {
		if e.cfg.StartWatcher {
			e.ConnWatcherStopped = false
			go e.startWatcher(ctx)
		} else {
			// Manually start the connection once to establish connection
			_ = e.watcher(ctx)
		}
	}

	// Queue worker will be started only if needed. If it won't be
	// started then queueWorkerStopped flag forced to true.
	if e.cfg.StartQueueWorker && e.queueWorkerStopped {
		e.queueWorkerStopped = false
		go e.startQueueWorker(ctx)
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established and database migrations will be applied
// (or rolled back).
func (e *Enity) WaitForEstablishing(ctx context.Context) {
	for {
		e.migratedMutex.Lock()
		migrated := e.migrated
		e.migratedMutex.Unlock()

		if e.Conn != nil && migrated {
			break
		}

		e.GetLogger(ctx).Debug().Msg("enity isn't ready")
		time.Sleep(time.Millisecond * 100)
	}
}

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) {
	e.GetLogger(ctx).Info().Msg("starting connection watcher")

	ticker := time.NewTicker(e.cfg.Timeout)

	// First start - manually.
	_ = e.watcher(ctx)

	// Then - every ticker tick.
	for range ticker.C {
		if e.watcher(ctx) {
			break
		}
	}

	ticker.Stop()
	e.GetLogger(ctx).Info().Msg("connection watcher stopped and connection to database was shutted down")
	e.ConnWatcherStopped = true
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
func (e *Enity) ping(ctx context.Context) error {
	if e.Conn == nil {
		return nil
	}

	if err := e.Conn.PingContext(ctx); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

// Connection watcher itself.
func (e *Enity) watcher(ctx context.Context) bool {
	// If we're shutting down - stop connection watcher.
	if e.WeAreShuttingDown {
		_ = e.shutdown(ctx)
		return true
	}

	if err := e.ping(ctx); err != nil {
		e.GetLogger(ctx).Error().Err(err).Msg("database connection lost")
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if e.Conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection to database...")
		// Connect to database.
		dbConn, err := sqlx.Connect("postgres", e.cfg.ComposeDSN())
		if err == nil {
			e.GetLogger(ctx).Info().Msg("database connection established")
			e.Conn = dbConn

			// Migrate database.
			e.Migrate(ctx)

			// Set connection pooling options.
			e.SetConnPoolLifetime(e.cfg.MaxConnectionLifetime)
			e.SetConnPoolLimits(e.cfg.MaxIdleConnections, e.cfg.MaxOpenedConnections)
			return false
		}

		if !e.cfg.StartWatcher {
			e.GetLogger(ctx).Fatal().Err(err).Msgf("failed to connect to database")
			return true
		}

		e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to database, reconnect after %d seconds", e.cfg.Timeout)
		return true
	}

	return false
}

func (e *Enity) setMigrationFlag() {
	e.migratedMutex.Lock()
	e.migrated = true
	// After successful migration we should not attempt to migrate it
	// again if connection to database was re-established.
	e.cfg.Migrate.Action = actionNothing
	e.migratedMutex.Unlock()
}

func (c *Enity) waitConn() {
	for {
		if c.Conn != nil {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
}

func (e *Enity) getCurrentDBVersion(ctx context.Context) int64 {
	migrationsLog := e.GetLogger(ctx).With().Str("subsystem", "database migrations").Logger()

	currentDBVersion, gooseerr := goose.GetDBVersion(e.Conn.DB)
	if gooseerr != nil {
		migrationsLog.Fatal().Err(gooseerr).Msg("failed to get database version")
	}

	return currentDBVersion
}

func (e *Enity) migrate(ctx context.Context, currentDBVersion int64) error {
	var err error

	switch {
	case e.cfg.Migrate.Action == actionUp && e.cfg.Migrate.Count == 0:
		e.GetLogger(ctx).Info().Msg("applying all unapplied migrations...")

		err = goose.Up(e.Conn.DB, e.cfg.Migrate.Directory)
	case e.cfg.Migrate.Action == actionUp && e.cfg.Migrate.Count != 0:
		newVersion := currentDBVersion + e.cfg.Migrate.Count

		e.GetLogger(ctx).Info().Int64("new version", newVersion).Msg("migrating database to specific version")

		err = goose.UpTo(e.Conn.DB, e.cfg.Migrate.Directory, newVersion)
	case e.cfg.Migrate.Action == actionDown && e.cfg.Migrate.Count == 0:
		e.GetLogger(ctx).Info().Msg("downgrading database to zero state, you'll need to re-apply migrations!")

		err = goose.DownTo(e.Conn.DB, e.cfg.Migrate.Directory, 0)

		e.GetLogger(ctx).Fatal().Msg("database downgraded to zero state, you have to re-apply migrations")
	case e.cfg.Migrate.Action == actionDown && e.cfg.Migrate.Count != 0:
		newVersion := currentDBVersion - e.cfg.Migrate.Count

		e.GetLogger(ctx).Info().Int64("new version", newVersion).Msg("downgrading database to specific version")

		err = goose.DownTo(e.Conn.DB, e.cfg.Migrate.Directory, newVersion)
	default:
		e.GetLogger(ctx).Fatal().
			Str("action", e.cfg.Migrate.Action).
			Int64("count", e.cfg.Migrate.Count).
			Msg("unsupported set of migration parameters, cannot continue")
	}

	return err
}

func (c *Enity) migrateSchema(ctx context.Context) error {
	if c.cfg.Migrate.Schema == "" {
		return nil
	}

	_, err := c.Conn.ExecContext(ctx, c.Conn.Rebind("CREATE SCHEMA IF NOT EXISTS "+c.cfg.Migrate.Schema))

	return err
}

// Migrates database.
func (e *Enity) Migrate(ctx context.Context) {
	e.waitConn()
	migrationsLog := e.GetLogger(ctx).With().Str("subsystem", "database migrations").Logger()

	err := e.migrateSchema(ctx)
	if err != nil {
		migrationsLog.Error().Err(err).Msg("failed to execute schema migration")
		e.setMigrationFlag()

		return
	}

	// Ensuring that we're using right database dialect. Without that
	// errors like:
	//
	//   pq: relation "goose_db_version" already exists
	//
	// might appear when that relation actually exists.
	_ = goose.SetDialect("clickhouse")

	currentDBVersion := e.getCurrentDBVersion(ctx)
	migrationsLog.Debug().Int64("database version", currentDBVersion).Msg("current database version obtained")

	if err := e.migrate(ctx, currentDBVersion); err != nil {
		migrationsLog.Fatal().Err(err).Msg("failed to execute migration sequence")
	}

	migrationsLog.Info().Msg("database migrated successfully")
	e.setMigrationFlag()

	// Figure out was migrate-only mode requested?
	if e.cfg.Migrate.Only {
		migrationsLog.Warn().Msg("only database migrations was requested, shutting down")
		os.Exit(0)
	}

	migrationsLog.Info().Msg("migrate-only mode wasn't requested")
}

// MigrationInCode represents informational struct for database migration
// that was written as Go code.
// When using such migrations you should not use SQL migrations as such
// mix might fuck up everything. This might be changed in future.
type MigrationInCode struct {
	Name string
	Down func(tx *sql.Tx) error
	Up   func(tx *sql.Tx) error
}

func (c *Enity) RegisterMigration(migration *MigrationInCode) {
	goose.AddNamedMigration(migration.Name, migration.Up, migration.Down)
}

// SetSchema sets schema for migrations.
func (c *Enity) SetSchema(schema string) {
	c.cfg.Migrate.Schema = schema
	// Add dbname as prefix after approved pull request https://github.com/pressly/goose/pull/228
	goose.SetTableName("goose_db_version")
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

func (c *Enity) AppendToQueue(queueItem *QueueItem) {
	defer c.queueMutex.Unlock()
	c.queueMutex.Lock()
	c.queue = append(c.queue, queueItem)
}

func (c *Enity) recreateQueue() {
	defer c.queueMutex.Unlock()
	c.queueMutex.Lock()
	c.queue = make([]*QueueItem, 0, 10240)
}

// Queue worker goroutine entry point.
func (e *Enity) startQueueWorker(ctx context.Context) {
	e.GetLogger(ctx).Info().Msg("starting queue worker")

	ticker := time.NewTicker(e.cfg.QueueWorkerTimeout)
	e.recreateQueue()

	e.queueMutex = &sync.Mutex{}

	for range ticker.C {
		if e.workWithQueue(ctx) {
			break
		}
	}

	ticker.Stop()
	e.GetLogger(ctx).Info().Msg("queue worker stopped")
	e.queueWorkerStopped = true
}

func (e *Enity) workWithQueue(ctx context.Context) bool {
	// We should stop on shutdown.
	if e.WeAreShuttingDown {
		return true
	}

	// Do nothing if connection wasn't yet established.
	if e.Conn == nil {
		e.GetLogger(ctx).Debug().Msg("connection to database wasn't established, do nothing")
		return false
	}

	e.queueMutex.Lock()
	var queriesToProcess []*QueueItem
	queriesToProcess = append(queriesToProcess, e.queue...)
	e.queueMutex.Unlock()
	e.recreateQueue()

	if e.WeAreShuttingDown {
		if len(queriesToProcess) == 0 {
			return true
		}

		e.GetLogger(ctx).Warn().Int("items in queue", len(queriesToProcess)).Msg("still has items in queue to process, delaying shutdown")
	}

	e.GetLogger(ctx).Debug().Int("items to process", len(queriesToProcess)).Msg("got queries to process")

	if len(queriesToProcess) == 0 {
		e.GetLogger(ctx).Debug().Msg("nothing to process, skipping iteration")
		return false
	}

	tx, err := e.Conn.Beginx()
	if err != nil {
		e.GetLogger(ctx).Error().
			Err(err).
			Msg("failed to start sql transaction; items will be pushed back to queue")
		e.queueMutex.Lock()
		e.queue = append(e.queue, queriesToProcess...)
		e.queueMutex.Unlock()

		return false
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

			_, err1 := tx.NamedExec(item.Query, item.Param)
			if err1 != nil {
				// Maybe write problematic queries somewhere.
				e.GetLogger(ctx).Error().Err(err1).Msg("failed to execute query!")
			}
		} else {
			e.GetLogger(ctx).Warn().Msgf("this item already present in database: %+v", item.Param)
		}
	}

	if err := tx.Commit(); err != nil {
		// What to do with items?
		e.GetLogger(ctx).Err(err).Msg("failed to commit transaction to database, rolling back...")

		errBack := tx.Rollback()
		if errBack != nil {
			e.GetLogger(ctx).Error().Err(errBack).Msg("failed to rollback failed transaction, expect database inconsistency!")
		}
	} else {
		e.GetLogger(ctx).Info().Msg("sql transaction committed")
	}

	if waitForFlush {
		waitForFlushChan <- true

		e.queueMutex.Lock()
		e.queue = append(queriesToProcess[reAddRangeStart:], e.queue...)
		e.queueMutex.Unlock()
	}

	return false
}

func (e *Enity) getDBStats() error {
	if e.Conn == nil {
		return nil
	}

	dbStats := e.Conn.DB.Stats()

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_open_connection"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_open_connection",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq open connection right now"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.OpenConnections))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_max_open_connection"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_max_open_connection",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq max open connection"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxOpenConnections))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_in_use"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_in_use",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq connection in use right now"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.InUse))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_wait_duration"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_wait_duration",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq wait duration"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.WaitDuration))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_max_idle_closed"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_max_idle_closed",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq max idle closed"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxIdleClosed))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_max_life_time_closed"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_max_life_time_closed",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq max life time closed"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxLifetimeClosed))
		},
	}

	// nolint : dupl
	e.Metrics[e.GetFullName()+"_ch_idle"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_ch_idle",
				Help: utils.JoinStrings(" ", e.GetFullName(), "pq idle"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.Idle))
		},
	}

	return nil
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
				err := e.ping(ctx)
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}

	if err := e.getDBStats(); err != nil {
		return nil, errors.Wrap(err, "get db stats")
	}
	return e.Metrics, nil
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	e.ReadyHandlers[strings.ToUpper(e.GetFullName()+"_notfailed")] = func() (bool, string) {
		if e.Conn == nil {
			return false, "not connected"
		}

		if err := e.ping(ctx); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return e.ReadyHandlers, nil
}
