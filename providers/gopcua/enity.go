package gopcua

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "gopcua"

var ErrFindSuitableEndpoint = errors.New("failed to find suitable endpoint")

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

	e := &Enity{
		MetricsStorage:    base.NewMetricsStorage(),
		ReadyCheckStorage: base.NewReadyCheckStorage(),
		Enity:             baseEnity,
		config:            config.SetDefault(),
	}

	if err := e.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	if err := e.buildReadyHandlers(ctx); err != nil {
		return nil, errors.Wrap(err, "build ready handlers")
	}

	return e, nil
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
func (e *Enity) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	logger := e.GetLogger(ctx)

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if e.conn != nil {
		return nil
	}
	// Connect to opc ua.
	endpoints, err := opcua.GetEndpoints(e.config.DSN)
	if err != nil {
		return errors.Wrap(err, "get endpoints")
	}

	ep := opcua.SelectEndpoint(endpoints, e.config.SecurityPolicy, ua.MessageSecurityModeFromString(e.config.SecurityMode))
	if ep == nil {
		return ErrFindSuitableEndpoint
	}

	logger.Info().Msgf("%s %s", ep.SecurityPolicyURI, ep.SecurityMode)

	opts := []opcua.Option{
		opcua.SecurityPolicy(e.config.SecurityPolicy),
		opcua.SecurityModeString(e.config.SecurityPolicy),
		opcua.CertificateFile(e.config.CertificateFile),
		opcua.PrivateKeyFile(e.config.PrivateKeyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	opcUAConn := opcua.NewClient(e.config.DSN, opts...)
	if err := opcUAConn.Connect(ctx); err != nil {
		return errors.Wrap(err, "connect opc ua")
	}

	e.conn = opcUAConn

	logger.Info().Msg("opc ua connection established")

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.IsWatcherStopped() {
		e.SetWatcher(false)
		errorGroup.Go(func() error {
			return e.startWatcher(ctx)
		})
	}

	return nil
}

// SubscribeOptions describes interface for subscriber
type Subscriber interface {
	Consume(ctx context.Context, data *monitor.DataChangeMessage) error
}

// Subscribe to channel for receiving message
func (e *Enity) Subscribe(ctx context.Context, errorGroup *errgroup.Group, subscriber Subscriber) error {
	if err := e.initSubscription(ctx); err != nil {
		return errors.Wrap(err, "init subscription")
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
		return errors.Wrap(err, "new node monitor")
	}

	e.ch = make(chan *monitor.DataChangeMessage, e.config.QueueSize)
	sub, err := m.ChanSubscribe(ctx, &opcua.SubscriptionParameters{Interval: e.config.Interval}, e.ch)
	if err != nil {
		return errors.Wrap(err, "chan subscribe")
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

	if _, err := e.conn.GetEndpoints(); err != nil {
		return errors.Wrap(err, "get endpoints")
	}
	return nil
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
