package pq

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	ctx     context.Context
	log     zerolog.Logger
	name    string
	options *Config

	Conn *sqlx.DB

	// Name of schema in database
	schemaName string

	migratedMutex sync.Mutex

	queue      []*QueueItem
	queueMutex *sync.Mutex

	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool
	queueWorkerStopped bool

	// Moved here for memory's sake. Guarded by mutex above.
	migrated bool

	// Metrics
	stats.Service
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, opts interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	conn := &Enity{
		name: name,
		ctx:  logger.Registrate(ctx),
	}

	logger.Get(conn.ctx).GetLogger(db.ProvidersName, nil).Info().Msgf("initializing enity " + name + "...")
	conn.log = logger.Get(conn.ctx).GetLogger(db.ProvidersName, nil).With().Str("connection", name).Logger()

	// We should not attempt to establish connection if passed options
	// isn't OUR options.
	var ok bool
	conn.options, ok = opts.(*Config)
	if !ok {
		return nil, db.ErrInvalidConnectionOptionsPointer(Config{})
	}

	conn.options.Validate()

	conn.name = name
	// Default schema is "production"
	conn.SetSchemaName("production")

	conn.connWatcherStopped = true
	conn.queueWorkerStopped = true

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (conn *Enity) Shutdown() error {
	conn.log.Info().Msg("Shutting down database connection watcher and queue worker")
	conn.weAreShuttingDown = true

	for {
		if conn.queueWorkerStopped && conn.connWatcherStopped {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
	conn.log.Info().Msg("Connection shutted down")

	return nil
}

// SetConnPoolLifetime sets connection lifetime.
func (conn *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	conn.options.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if conn.Conn != nil {
		conn.Conn.SetConnMaxLifetime(connMaxLifetime)
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (conn *Enity) SetConnPoolLimits(maxIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	conn.options.MaxIdleConnections = maxIdleConnections
	conn.options.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if conn.Conn != nil {
		conn.Conn.SetMaxIdleConns(maxIdleConnections)
		conn.Conn.SetMaxOpenConns(maxOpenedConnections)
	}
}

// SetPoolLimits sets connection pool limits.
func (conn *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	conn.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	conn.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (conn *Enity) Start() error {
	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if conn.connWatcherStopped {
		if conn.options.StartWatcher {
			conn.connWatcherStopped = false
			go conn.startWatcher()
		} else {
			// Manually start the connection once to establish connection
			_ = conn.watcher()
		}
	}

	// Queue worker will be started only if needed. If it won't be
	// started then queueWorkerStopped flag forced to true.
	if conn.options.StartQueueWorker && conn.queueWorkerStopped {
		conn.queueWorkerStopped = false
		go conn.startQueueWorker()
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established and database migrations will be applied
// (or rolled back).
func (conn *Enity) WaitForEstablishing() {
	for {
		conn.migratedMutex.Lock()
		migrated := conn.migrated
		conn.migratedMutex.Unlock()

		if conn.Conn != nil && migrated {
			break
		}

		conn.log.Debug().Msg("Connection isn't ready - not yet established or database migration in progress")
		time.Sleep(time.Millisecond * 100)
	}
}

// createMutexConnect initialize connection for mutex
func (conn *Enity) createMutexConnect() (*sqlx.DB, error) {
	dsn := conn.options.DSN
	if conn.options.Options != "" {
		dsn += "?" + conn.options.Options
	}

	// Connect to database.
	dbConn, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Only one connect in pool for mutex
	dbConn.SetMaxOpenConns(1)

	return dbConn, nil
}

// NewMutex create new database mutex
func (conn *Enity) NewMutex(checkInterval time.Duration) (*Mutex, error) {
	dbConn, err := conn.createMutexConnect()
	if err != nil {
		return nil, err
	}

	return NewMutex(&dbConn, checkInterval)
}

// NewMutexByID create new database mutex with selected id
func (conn *Enity) NewMutexByID(lockID interface{}, checkInterval time.Duration) (*Mutex, error) {
	dbConn, err := conn.createMutexConnect()
	if err != nil {
		return nil, err
	}

	return NewMutexByID(&dbConn, lockID, checkInterval)
}

// GetMetrics return map of the metrics from database connection
func (conn *Enity) GetMetrics(prefix string) stats.MapMetricsOptions {
	_ = conn.Service.GetMetrics(prefix)
	conn.Metrics[prefix+"_"+conn.name+"_status"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + conn.name + "_status",
				Help: prefix + " " + conn.name + " status link to " + utils.RedactedDSN(conn.options.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			if conn.Conn != nil {
				err := conn.Conn.Ping()
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}
	return conn.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (conn *Enity) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	_ = conn.Service.GetReadyHandlers(prefix)
	conn.ReadyHandlers[strings.ToUpper(prefix+"_"+conn.name+"_notfailed")] = func() (bool, string) {
		if conn.Conn == nil {
			return false, "Not connected"
		}

		if err := conn.Conn.Ping(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return conn.ReadyHandlers
}
