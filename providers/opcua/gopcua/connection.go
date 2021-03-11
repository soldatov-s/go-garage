package gopcua

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
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
	// OPC UA connection
	Conn *opcua.Client
	// Subscription
	Subscription *monitor.Subscription

	ctx  context.Context
	log  zerolog.Logger
	name string
	cfg  *Config
	ch   chan *monitor.DataChangeMessage

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

	conn.log.Info().Msgf("initializing enity " + name + "...")

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (c *Enity) Shutdown() error {
	c.log.Info().Msg("shutting down opc ua connection watcher and queue worker")
	c.weAreShuttingDown = true

	if c.cfg.StartWatcher {
		for {
			if c.connWatcherStopped {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else {
		if c.Subscription != nil {
			c.Subscription.Unsubscribe()
		}
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

// Subscribe to channel for receiving message
func (c *Enity) Subscribe(options *SubscribeOptions) error {
	if err := c.initSubscription(); err != nil {
		return err
	}

	go c.subscribe(options)

	return nil
}

func (c *Enity) subscribe(options *SubscribeOptions) {
	for res := range c.ch {
		if res.Error != nil {
			c.log.Error().Err(res.Error).Msgf("failed to get notification")
			continue
		}
		if options != nil {
			if err1 := options.ConsumeHndl(res); err1 != nil {
				c.log.Error().Msgf("can't consume. error: %s", err1)
				continue
			}
		}
		time.Sleep(c.cfg.Interval)
	}
}

func (c *Enity) initSubscription() error {
	if c.Subscription != nil {
		return nil
	}

	m, err := monitor.NewNodeMonitor(c.Conn)
	if err != nil {
		return err
	}

	c.ch = make(chan *monitor.DataChangeMessage, 16)
	sub, err := m.ChanSubscribe(c.ctx, &opcua.SubscriptionParameters{Interval: c.cfg.Interval}, c.ch)
	if err != nil {
		return err
	}

	c.log.Info().Msgf("created subscription with id %d", sub.SubscriptionID())

	c.Subscription = sub
	return nil
}

func (c *Enity) SubscribeNodeID(nodeID string) error {
	c.log.Debug().Msgf("subscribe nodeID %s", nodeID)
	return c.Subscription.AddNodes(nodeID)
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
