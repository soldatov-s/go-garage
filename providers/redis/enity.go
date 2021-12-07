package redis

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
	rediscache "github.com/soldatov-s/go-garage/providers/redis/cache"
	"github.com/soldatov-s/go-garage/providers/redis/rejson"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "redis"

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	*base.ReadyCheckStorage
	conn   *rejson.Client
	config *Config
	caches map[string]*rediscache.Cache
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
		caches:            make(map[string]*rediscache.Cache),
	}

	if err := e.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	if err := e.buildReadyHandlers(ctx); err != nil {
		return nil, errors.Wrap(err, "build ready handlers")
	}

	return e, nil
}

func (e *Enity) GetConn() *rejson.Client {
	return e.conn
}

func (e *Enity) GetConfig() *Config {
	return e.config
}

func (e *Enity) AddCache(ctx context.Context, config *rediscache.Config) (*rediscache.Cache, error) {
	name := config.KeyPrefix
	config.GlobalKeyPrefix = e.config.GlobalCacheKeyPrefix
	if _, ok := e.caches[name]; ok {
		return nil, base.ErrConflictName
	}

	cache, err := rediscache.NewCache(ctx, e.GetFullName()+"_"+name, config, &e.conn)
	if err != nil {
		return nil, errors.Wrap(err, "new consumer")
	}

	e.caches[name] = cache
	if err := e.MetricsStorage.GetMetrics().Append(cache.GetMetrics()); err != nil {
		return nil, errors.Wrap(err, "append metrics")
	}

	return cache, nil
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

// SetConnPoolLifetime sets connection lifetime.
func (e *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	e.config.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if e.conn != nil {
		e.conn.Options().MaxConnAge = connMaxLifetime
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (e *Enity) SetConnPoolLimits(minIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	e.config.MinIdleConnections = minIdleConnections
	e.config.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if e.conn != nil {
		e.conn.Options().MinIdleConns = minIdleConnections
		e.conn.Options().PoolSize = maxOpenedConnections
	}
}

// SetPoolLimits sets connection pool limits.
func (e *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	e.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	e.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (e *Enity) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	logger := e.GetLogger(ctx)

	if e.conn != nil {
		return nil
	}
	logger.Info().Msg("establishing connection ...")

	// Connect to database.
	connOptions, err := e.config.Options()
	if err != nil {
		return errors.Wrap(err, "parse options")
	}

	// Set connection pooling options.
	connOptions.MaxConnAge = e.config.MaxConnectionLifetime
	connOptions.MinIdleConns = e.config.MinIdleConnections
	connOptions.PoolSize = e.config.MaxOpenedConnections

	conn := redis.NewClient(connOptions)
	if _, errPing := conn.Ping(ctx).Result(); errPing != nil {
		return errors.Wrap(errPing, "connect to enity")
	}

	e.conn = rejson.ExtendClient(conn)

	logger.Info().Msg("connection established")

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

	if _, err := e.conn.Ping(ctx).Result(); err != nil {
		return errors.Wrap(err, "ping connection")
	}

	return nil
}

// NewMutex create new redis mutex
func (e *Enity) NewMutex(opts ...MutexOption) (*Mutex, error) {
	return NewMutex(e.conn, opts...)
}

// GetMetrics return map of the metrics from cache connection
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
