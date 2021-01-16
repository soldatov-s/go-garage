package mongo

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	// Metrics
	stats.Service

	// Connection to database
	Conn *mongo.Client
	ctx  context.Context
	log  zerolog.Logger
	name string
	cfg  *Config

	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool
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
	}

	conn.log.Info().Msgf("initializing connection...")

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (c *Enity) Shutdown() error {
	c.log.Info().Msg("shutting down database connection")
	c.weAreShuttingDown = true

	if c.cfg.StartWatcher {
		for {
			if c.connWatcherStopped {
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

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established and database migrations will be applied
// (or rolled back).
func (c *Enity) WaitForEstablishing() {
	for {
		if c.Conn != nil {
			break
		}

		c.log.Debug().Msg("connection isn't ready")
		time.Sleep(time.Millisecond * 100)
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
