package dbgraph

import (
	"context"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const ProviderName = "dbgraph"

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	*base.ReadyCheckStorage
	grpcConn *grpc.ClientConn
	conn     *dgo.Dgraph
	config   *Config
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

func (e *Enity) GetConn() *dgo.Dgraph {
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

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) {
	e.GetLogger(ctx).Info().Msg("starting connection watcher")

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
	e.GetLogger(ctx).Info().Msg("connection watcher stopped and connection to database was shutted down")
	e.SetWatcher(true)
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}
	e.GetLogger(ctx).Info().Msg("closing connection...")

	if err := e.grpcConn.Close(); err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	e.grpcConn = nil

	return nil
}

// Pinging connection if it's alive (or we think so).
func (e *Enity) Ping(ctx context.Context) error {
	if e.grpcConn == nil {
		return nil
	}

	connState := e.grpcConn.GetState()
	if connState == connectivity.Shutdown || connState == connectivity.TransientFailure {
		return errors.New("connection not alive")
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
		e.GetLogger(ctx).Error().Err(err).Msg("database connection lost")
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if e.conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection to database...")
		// Dial a gRPC connection. The address to dial to can be configured when
		// setting up the dgraph cluster.
		d, err := grpc.Dial(e.config.DSN, grpc.WithInsecure())
		if err == nil {
			e.GetLogger(ctx).Info().Msg("database connection established")
			e.grpcConn = d
			e.conn = dgo.NewDgraphClient(
				api.NewDgraphClient(d),
			)

			return false
		}

		if !e.config.StartWatcher {
			e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to database")
			return true
		}

		e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to database, reconnect after %d seconds", e.config.Timeout)
		return true
	}

	return false
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
