package dbgraph

import (
	"context"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	grpcConn *grpc.ClientConn
	Conn     *dgo.Dgraph
	cfg      *Config
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
// successfully established and database migrations will be applied
// (or rolled back).
func (e *Enity) WaitForEstablishing(ctx context.Context) {
	for {
		if e.Conn != nil {
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

	err := e.grpcConn.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close connection")
	}

	e.grpcConn = nil

	return nil
}

// Pinging connection if it's alive (or we think so).
func (e *Enity) ping(ctx context.Context) error {
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
		// Dial a gRPC connection. The address to dial to can be configured when
		// setting up the dgraph cluster.
		d, err := grpc.Dial(e.cfg.DSN, grpc.WithInsecure())
		if err == nil {
			e.GetLogger(ctx).Info().Msg("database connection established")
			e.grpcConn = d
			e.Conn = dgo.NewDgraphClient(
				api.NewDgraphClient(d),
			)

			return false
		}

		if !e.cfg.StartWatcher {
			e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to database")
			return true
		}

		e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to database, reconnect after %d seconds", e.cfg.Timeout)
		return true
	}

	return false
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) GetMetrics(ctx context.Context) base.MapMetricsOptions {
	e.Metrics.AddNewMetricGauge(
		e.GetFullName(),
		"status",
		utils.JoinStrings(" ", "status link to", utils.RedactedDSN(e.cfg.DSN)),
		func() float64 {
			if e.Conn != nil {
				err := e.ping(ctx)
				if err == nil {
					return 1
				}
			}
			return 0
		},
	)

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
