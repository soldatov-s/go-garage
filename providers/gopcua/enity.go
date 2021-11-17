package gopcua

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "gopcua"

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	*base.ReadyCheckStorage
	config *Config
	// OPC UA connection
	conn *opcua.Client
	// subscription
	subscription *monitor.Subscription
	ch           chan *monitor.DataChangeMessage
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

	enity := &Enity{
		MetricsStorage:    base.NewMetricsStorage(),
		ReadyCheckStorage: base.NewReadyCheckStorage(),
		Enity:             baseEnity,
		config:            config.SetDefault(),
	}
	if err := enity.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	if err := enity.buildReadyHandlers(ctx); err != nil {
		return nil, errors.Wrap(err, "build ready handlers")
	}

	return enity, nil
}

func (e *Enity) GetSubscription() *monitor.Subscription {
	return e.subscription
}

func (e *Enity) GetConn() *opcua.Client {
	return e.conn
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
			if e.IsWatcherStopped() {
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
	if e.IsWatcherStopped() {
		if e.config.StartWatcher {
			e.SetWatcher(false)
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
		if e.conn != nil {
			break
		}

		e.GetLogger(ctx).Debug().Msg("enity isn't ready")
		time.Sleep(time.Millisecond * 100)
	}
}

// SubscribeOptions describes interface for subscriber
type Subscriber interface {
	Consume(ctx context.Context, data *monitor.DataChangeMessage) error
}

// Subscribe to channel for receiving message
func (e *Enity) Subscribe(ctx context.Context, errorGroup *errgroup.Group, subscriber Subscriber) error {
	if err := e.initSubscription(ctx); err != nil {
		return err
	}

	errorGroup.Go(func() error {
		return e.subscribe(ctx, subscriber)
	})

	return nil
}

func (e *Enity) subscribe(ctx context.Context, subscriber Subscriber) error {
	for {
		select {
		case <-ctx.Done():
			e.GetLogger(ctx).Info().Msg("subscribe stopped")
			return ctx.Err()
		case res := <-e.ch:
			if res.Error != nil {
				e.GetLogger(ctx).Error().Err(res.Error).Msgf("failed to get notification")
				continue
			}
			if err1 := subscriber.Consume(ctx, res); err1 != nil {
				e.GetLogger(ctx).Error().Msgf("can't consume. error: %s", err1)
			}
		default:
			time.Sleep(e.config.Interval)
		}
	}
}

func (e *Enity) initSubscription(ctx context.Context) error {
	if e.subscription != nil {
		return nil
	}

	m, err := monitor.NewNodeMonitor(e.conn)
	if err != nil {
		return err
	}

	e.ch = make(chan *monitor.DataChangeMessage, e.config.QueueSize)
	sub, err := m.ChanSubscribe(ctx, &opcua.SubscriptionParameters{Interval: e.config.Interval}, e.ch)
	if err != nil {
		return err
	}

	e.GetLogger(ctx).Info().Msgf("created subscription with id %d", sub.SubscriptionID())

	e.subscription = sub
	return nil
}

func (e *Enity) SubscribeNodeID(ctx context.Context, nodeID string) error {
	e.GetLogger(ctx).Debug().Msgf("subscribe nodeID %s", nodeID)
	return e.subscription.AddNodes(nodeID)
}

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) {
	e.GetLogger(ctx).Info().Msg("starting watcher...")
	ticker := time.NewTicker(e.config.Timeout)

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
	e.SetWatcher(true)
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

	if _, err := e.conn.GetEndpoints(); err != nil {
		return errors.Wrap(err, "get endpoints")
	}
	return nil
}

// Connection watcher itself.
func (e *Enity) watcher(ctx context.Context) bool {
	logger := zerolog.Ctx(ctx)
	// If we're shutting down - stop connection watcher.
	if e.IsShuttingDown() {
		if err := e.shutdown(ctx); err != nil {
			logger.Err(err).Msg("shutdown")
		}
		return true
	}

	if err := e.Ping(ctx); err != nil {
		e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if e.conn != nil {
		return false
	}
	e.GetLogger(ctx).Info().Msg("establishing connection to opc ua...")
	// Connect to opc ua.
	endpoints, err := opcua.GetEndpoints(e.config.DSN)
	if err != nil {
		e.GetLogger(ctx).Fatal().Err(err).Msg("failed to get endpoints")
	}

	ep := opcua.SelectEndpoint(endpoints, e.config.SecurityPolicy, ua.MessageSecurityModeFromString(e.config.SecurityMode))
	if ep == nil {
		e.GetLogger(ctx).Fatal().Err(err).Msg("failed to find suitable endpoint")
	} else {
		e.GetLogger(ctx).Info().Msgf("%s %s", ep.SecurityPolicyURI, ep.SecurityMode)
	}

	opts := []opcua.Option{
		opcua.SecurityPolicy(e.config.SecurityPolicy),
		opcua.SecurityModeString(e.config.SecurityPolicy),
		opcua.CertificateFile(e.config.CertificateFile),
		opcua.PrivateKeyFile(e.config.PrivateKeyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	opcUAConn := opcua.NewClient(e.config.DSN, opts...)
	err = opcUAConn.Connect(ctx)
	if err == nil {
		e.GetLogger(ctx).Info().Msg("opc ua connection established")
		e.conn = opcUAConn

		return false
	}

	if !e.config.StartWatcher {
		e.GetLogger(ctx).Fatal().Err(err).Msgf("connect to opc ua")
		return true
	}

	e.GetLogger(ctx).Error().Err(err).Msgf("connect to opc ua, reconnect after %d seconds", e.config.Timeout)
	return true
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) buildMetrics(_ context.Context) error {
	fullName := e.GetFullName()
	redactedDSN, err := stringsx.RedactedDSN(e.config.DSN)
	if err != nil {
		return errors.Wrap(err, "redacted dsn")
	}
	help := stringsx.JoinStrings(" ", "status link to", redactedDSN)
	metricFunc := func(ctx context.Context) (float64, error) {
		if e.conn != nil {
			err := e.Ping(ctx)
			if err == nil {
				return 1, nil
			}
		}
		return 0, nil
	}
	if _, err := e.MetricsStorage.GetMetrics().AddGauge(fullName, "status", help, metricFunc); err != nil {
		return errors.Wrap(err, "add gauge metric")
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
