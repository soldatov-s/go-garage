package pq

import (
	"context"
	"database/sql"
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
	// Metrics
	stats.Service
	// DB connection
	Conn *sqlx.DB

	ctx           context.Context
	log           zerolog.Logger
	name          string
	cfg           *Config
	migratedMutex sync.Mutex

	// Queue for bulk operations
	queue              []*QueueItem
	queueMutex         *sync.Mutex
	queueWorkerStopped bool
	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool
	// Migration flag
	migrated bool
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, cfg interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	// Checking that passed config is OUR.
	if _, ok := cfg.(*Config); !ok {
		return nil, db.ErrNotConfigPointer(Config{})
	}

	conn := &Enity{
		name:               name,
		ctx:                logger.Registrate(ctx),
		log:                logger.Get(ctx).GetLogger(db.ProvidersName, nil).With().Str("connection", name).Logger(),
		cfg:                cfg.(*Config).SetDefault(),
		connWatcherStopped: true,
		queueWorkerStopped: true,
	}

	conn.log.Info().Msgf("initializing enity " + name + "...")

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (c *Enity) Shutdown() error {
	c.log.Info().Msg("shutting down database connection watcher and queue worker")
	c.weAreShuttingDown = true

	if c.cfg.StartWatcher {
		for {
			if c.queueWorkerStopped && c.connWatcherStopped {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else {
		c.shutdown()
	}

	c.log.Info().Msg("connection shutted down")

	return nil
}

// SetConnPoolLifetime sets connection lifetime.
func (c *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	c.cfg.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if c.Conn != nil {
		c.Conn.SetConnMaxLifetime(connMaxLifetime)
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (c *Enity) SetConnPoolLimits(maxIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	c.cfg.MaxIdleConnections = maxIdleConnections
	c.cfg.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if c.Conn != nil {
		c.Conn.SetMaxIdleConns(maxIdleConnections)
		c.Conn.SetMaxOpenConns(maxOpenedConnections)
	}
}

// SetPoolLimits sets connection pool limits.
func (c *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	c.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	c.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (c *Enity) Start() error {
	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if c.connWatcherStopped {
		if c.cfg.StartWatcher {
			c.connWatcherStopped = false
			go c.startWatcher()
		} else {
			// Manually start the connection once to establish connection
			_ = c.watcher()
		}
	}

	// Queue worker will be started only if needed. If it won't be
	// started then queueWorkerStopped flag forced to true.
	if c.cfg.StartQueueWorker && c.queueWorkerStopped {
		c.queueWorkerStopped = false
		go c.startQueueWorker()
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established and database migrations will be applied
// (or rolled back).
func (c *Enity) WaitForEstablishing() {
	for {
		c.migratedMutex.Lock()
		migrated := c.migrated
		c.migratedMutex.Unlock()

		if c.Conn != nil && migrated {
			break
		}

		c.log.Debug().Msg("connection isn't ready")
		time.Sleep(time.Millisecond * 100)
	}
}

// createMutexConnect initialize connection for mutex
func (c *Enity) createMutexConnect() (*sqlx.DB, error) {
	// Connect to database.
	dbConn, err := sqlx.Connect("postgres", c.cfg.ComposeDSN())
	if err != nil {
		return nil, err
	}

	// Only one connect in pool for mutex
	dbConn.SetMaxOpenConns(1)

	return dbConn, nil
}

// NewMutex create new database mutex
func (c *Enity) NewMutex(checkInterval time.Duration) (*Mutex, error) {
	dbConn, err := c.createMutexConnect()
	if err != nil {
		return nil, err
	}

	return NewMutex(dbConn, checkInterval)
}

// NewMutexByID create new database mutex with selected id
func (c *Enity) NewMutexByID(lockID int64, checkInterval time.Duration) (*Mutex, error) {
	dbConn, err := c.createMutexConnect()
	if err != nil {
		return nil, err
	}

	return NewMutexByID(dbConn, lockID, checkInterval)
}

func (c *Enity) getDBStats(prefix string) {
	var dbStats sql.DBStats
	if c.Conn != nil {
		dbStats = c.Conn.DB.Stats()
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_open_connection"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_open_connection",
				Help: prefix + " " + c.name + " pq open connection right now",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.OpenConnections))
		},
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_max_open_connection"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_max_open_connection",
				Help: prefix + " " + c.name + " pq max open connection",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxOpenConnections))
		},
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_in_use"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_in_use",
				Help: prefix + " " + c.name + " pq connection in use right now",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.InUse))
		},
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_wait_duration"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_wait_duration",
				Help: prefix + " " + c.name + " pq wait duration",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.WaitDuration))
		},
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_max_idle_closed"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_max_idle_closed",
				Help: prefix + " " + c.name + " pq max idle closed",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxIdleClosed))
		},
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_max_life_time_closed"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_max_life_time_closed",
				Help: prefix + " " + c.name + " pq max life time closed",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxLifetimeClosed))
		},
	}

	// nolint : dupl
	c.Metrics[prefix+"_"+c.name+"_pq_idle"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_pq_idle",
				Help: prefix + " " + c.name + " pq idle",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.Idle))
		},
	}
}

// GetMetrics return map of the metrics from database connection
func (c *Enity) GetMetrics(prefix string) stats.MapMetricsOptions {
	_ = c.Service.GetMetrics(prefix)
	c.Metrics[prefix+"_"+c.name+"_status"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_status",
				Help: prefix + " " + c.name + " status link to " + utils.RedactedDSN(c.cfg.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			if c.Conn != nil {
				err := c.ping()
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}

	c.getDBStats(prefix)
	return c.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (c *Enity) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	_ = c.Service.GetReadyHandlers(prefix)
	c.ReadyHandlers[strings.ToUpper(prefix+"_"+c.name+"_notfailed")] = func() (bool, string) {
		if c.Conn == nil {
			return false, "not connected"
		}

		if err := c.ping(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return c.ReadyHandlers
}
