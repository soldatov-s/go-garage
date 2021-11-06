package gopcua

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	cfg *Config
	// OPC UA connection
	Conn *opcua.Client
	// Subscription
	Subscription *monitor.Subscription
	ch           chan *monitor.DataChangeMessage
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

	return &Enity{EntityWithMetrics: e, cfg: config.SetDefault()}, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("shutting down")
	e.WeAreShuttingDown = true

	if e.cfg.StartWatcher {
		for {
			if e.ConnWatcherStopped {
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

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established.
func (e *Enity) WaitForEstablishing(ctx context.Context) {
	for {
		if e.Conn != nil {
			break
		}

		e.GetLogger(ctx).Debug().Msg("enity isn't ready")
		time.Sleep(time.Millisecond * 100)
	}
}

// Subscribe to channel for receiving message
func (e *Enity) Subscribe(ctx context.Context, options *SubscribeOptions) error {
	if err := e.initSubscription(ctx); err != nil {
		return err
	}

	go e.subscribe(ctx, options)

	return nil
}

func (e *Enity) subscribe(ctx context.Context, options *SubscribeOptions) {
	for res := range e.ch {
		if res.Error != nil {
			e.GetLogger(ctx).Error().Err(res.Error).Msgf("failed to get notification")
			continue
		}
		if options != nil {
			if err1 := options.ConsumeHndl(res); err1 != nil {
				e.GetLogger(ctx).Error().Msgf("can't consume. error: %s", err1)
				continue
			}
		}
		time.Sleep(e.cfg.Interval)
	}
}

func (e *Enity) initSubscription(ctx context.Context) error {
	if e.Subscription != nil {
		return nil
	}

	m, err := monitor.NewNodeMonitor(e.Conn)
	if err != nil {
		return err
	}

	e.ch = make(chan *monitor.DataChangeMessage, e.cfg.QueueSize)
	sub, err := m.ChanSubscribe(ctx, &opcua.SubscriptionParameters{Interval: e.cfg.Interval}, e.ch)
	if err != nil {
		return err
	}

	e.GetLogger(ctx).Info().Msgf("created subscription with id %d", sub.SubscriptionID())

	e.Subscription = sub
	return nil
}

func (e *Enity) SubscribeNodeID(ctx context.Context, nodeID string) error {
	e.GetLogger(ctx).Debug().Msgf("subscribe nodeID %s", nodeID)
	return e.Subscription.AddNodes(nodeID)
}

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) {
	e.GetLogger(ctx).Info().Msg("starting watcher...")
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
	e.GetLogger(ctx).Info().Msg("watcher stopped and enity was shutted down")
	e.ConnWatcherStopped = true
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.Conn == nil {
		return nil
	}

	e.GetLogger(ctx).Info().Msg("closing connection...")

	if err := e.Conn.Close(); err != nil {
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
	// TODO: write some code for ping opc ua service
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
		e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if e.Conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection to opc ua...")
		// Connect to opc ua.
		endpoints, err := opcua.GetEndpoints(e.cfg.DSN)
		if err != nil {
			e.GetLogger(ctx).Fatal().Err(err).Msg("failed to get endpoints")
		}

		ep := opcua.SelectEndpoint(endpoints, e.cfg.SecurityPolicy, ua.MessageSecurityModeFromString(e.cfg.SecurityMode))
		if ep == nil {
			e.GetLogger(ctx).Fatal().Err(err).Msg("failed to find suitable endpoint")
		}

		e.GetLogger(ctx).Info().Msgf("%s %s", ep.SecurityPolicyURI, ep.SecurityMode)

		opts := []opcua.Option{
			opcua.SecurityPolicy(e.cfg.SecurityPolicy),
			opcua.SecurityModeString(e.cfg.SecurityPolicy),
			opcua.CertificateFile(e.cfg.CertificateFile),
			opcua.PrivateKeyFile(e.cfg.PrivateKeyFile),
			opcua.AuthAnonymous(),
			opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		}

		opcUAConn := opcua.NewClient(e.cfg.DSN, opts...)
		err = opcUAConn.Connect(ctx)
		if err == nil {
			e.GetLogger(ctx).Info().Msg("opc ua connection established")
			e.Conn = opcUAConn

			return false
		}

		if !e.cfg.StartWatcher {
			e.GetLogger(ctx).Fatal().Err(err).Msgf("failed to connect to opc ua")
			return true
		}

		e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to opc ua, reconnect after %d seconds", e.cfg.Timeout)
		return true
	}

	return false
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) GetMetrics(ctx context.Context) base.MapMetricsOptions {
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

	return e.Metrics
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
