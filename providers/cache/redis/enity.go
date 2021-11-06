package redis

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
	"github.com/soldatov-s/go-garage/x/rejson"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	conn *rejson.Client
	cfg  *Config
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

// GetConn return enity connection
func (e *Enity) GetConn() *rejson.Client {
	return e.conn
}

// GetConfig return enity config
func (e *Enity) GetConfig() *Config {
	return e.cfg
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

// SetConnPoolLifetime sets connection lifetime.
func (e *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	e.cfg.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if e.conn != nil {
		e.conn.Options().MaxConnAge = connMaxLifetime
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (e *Enity) SetConnPoolLimits(minIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	e.cfg.MinIdleConnections = minIdleConnections
	e.cfg.MaxOpenedConnections = maxOpenedConnections

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
		if e.conn != nil {
			break
		}

		e.GetLogger(ctx).Debug().Msg("enity isn't ready")
		time.Sleep(time.Millisecond * 100)
	}
}

// Get item from cache by key.
func (e *Enity) Get(ctx context.Context, key string, value interface{}) error {
	cmdString := e.conn.Get(ctx, e.cfg.KeyPrefix+key)
	if _, err := cmdString.Result(); err != nil {
		return cache.ErrNotFoundInCache
	}

	if err := cmdString.Scan(value); err != nil {
		return errors.Wrap(err, "unmarshal value")
	}

	e.GetLogger(ctx).Debug().Msgf("get value by key %s from cache", key)
	return nil
}

// JSONGet item from cache by key.
func (e *Enity) JSONGet(ctx context.Context, key, path string, value interface{}) error {
	cmdString := e.conn.JSONGet(ctx, e.cfg.KeyPrefix+key, path)
	if _, err := cmdString.Result(); err != nil {
		return cache.ErrNotFoundInCache
	}

	if err := cmdString.Scan(value); err != nil {
		return errors.Wrap(err, "unmarshal value")
	}

	e.GetLogger(ctx).Debug().Msgf("jsonget value by key %s from cache, path %s", key, path)
	return nil
}

// Set item in cache by key.
func (e *Enity) Set(ctx context.Context, key string, value interface{}) error {
	if _, err := e.conn.Set(ctx, e.cfg.KeyPrefix+key, value, e.cfg.ClearTime).Result(); err != nil {
		return errors.Wrap(err, "set key")
	}

	e.GetLogger(ctx).Debug().Msgf("set key %s in cache, value %+v", key, value)
	return nil
}

// SetNX (Not eXist) item in cache by key.
func (e *Enity) SetNX(ctx context.Context, key string, value interface{}) error {
	if _, err := e.conn.SetNX(ctx, e.cfg.KeyPrefix+key, value, e.cfg.ClearTime).Result(); err != nil {
		return errors.Wrap(err, "setNX key")
	}

	e.GetLogger(ctx).Debug().Msgf("setnx key %s in cache, value %+v", key, value)
	return nil
}

// JSONSet item in cache by key.
func (e *Enity) JSONSet(ctx context.Context, key, path, json string) error {
	if _, err := e.conn.JSONSet(ctx, e.cfg.KeyPrefix+key, path, json).Result(); err != nil {
		return errors.Wrap(err, "JSONSet key")
	}

	if e.cfg.ClearTime > 0 {
		_, err := e.conn.Expire(ctx, e.cfg.KeyPrefix+key, e.cfg.ClearTime).Result()
		if err != nil {
			return errors.Wrap(err, "expire key")
		}
	}

	e.GetLogger(ctx).Debug().Msgf("jsonset key %s in cache, path %s, json %s", key, path, json)
	return nil
}

// JSONSetNX item in cache by key.
func (e *Enity) JSONSetNX(ctx context.Context, key, path, json string) error {
	if _, err := e.conn.JSONSet(ctx, e.cfg.KeyPrefix+key, path, json, "NX").Result(); err != nil {
		return errors.Wrap(err, "JSONSetNX key")
	}

	if e.cfg.ClearTime > 0 {
		_, err := e.conn.Expire(ctx, e.cfg.KeyPrefix+key, e.cfg.ClearTime).Result()
		if err != nil {
			return errors.Wrap(err, "expire key")
		}
	}

	e.GetLogger(ctx).Debug().Msgf("jsonsetnx key %s in cache, path %s, json %s", key, path, json)
	return nil
}

// Delete item from cache by key.
func (e *Enity) Delete(ctx context.Context, key string) error {
	if _, err := e.conn.Del(ctx, e.cfg.KeyPrefix+key).Result(); err != nil {
		return errors.Wrap(err, "del key")
	}

	e.GetLogger(ctx).Debug().Msgf("delete key %s from cache", key)
	return nil
}

// Size return count of item in cache
func (e *Enity) Size(ctx context.Context) (int, error) {
	length := 0
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = e.conn.Scan(ctx, cursor, e.cfg.KeyPrefix+"*", 10).Result()
		if err != nil {
			return -1, errors.Wrap(err, "scan keys")
		}

		length += len(keys)

		if cursor == 0 {
			break
		}
	}

	return length, nil
}

// Clear clear all items from selected connection.
func (e *Enity) Clear(ctx context.Context) error {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = e.conn.Scan(ctx, cursor, e.cfg.KeyPrefix+"*", 10).Result()
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			continue
		}

		pipe := e.conn.Pipeline()
		err = pipe.Del(ctx, keys...).Err()
		if err != nil {
			return err
		}

		_, err = pipe.Exec(e.conn.Context())
		if err != nil {
			return err
		}
		if cursor == 0 {
			break
		}
	}

	e.GetLogger(ctx).Debug().Msgf("delete all keys from cache")
	return nil
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
func (e *Enity) ping(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}

	if _, err := e.conn.Ping(ctx).Result(); err != nil {
		return errors.Wrap(err, "ping connection")
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
		e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
	}

	// If connection is nil - try to establish
	// connection.
	if e.conn == nil {
		e.GetLogger(ctx).Info().Msg("establishing connection ...")

		// Connect to database.
		connOptions, err := e.cfg.Options()
		if err != nil {
			e.GetLogger(ctx).Fatal().Err(err).Msgf("failed parse options: %s", err)
		}

		// Set connection pooling options.
		connOptions.MaxConnAge = e.cfg.MaxConnectionLifetime
		connOptions.MinIdleConns = e.cfg.MinIdleConnections
		connOptions.PoolSize = e.cfg.MaxOpenedConnections

		conn := redis.NewClient(connOptions)
		if conn.Ping(conn.Context()).Err() == nil {
			e.GetLogger(ctx).Info().Msg("connection established")
			e.conn = rejson.ExtendClient(conn)
			return false
		}
		if !e.cfg.StartWatcher {
			e.GetLogger(ctx).Fatal().Err(err).Msg("failed to connect")
		}

		e.GetLogger(ctx).Error().Err(err).Msgf("failed to connect to cache, will try to reconnect after %d seconds", e.cfg.Timeout)
	}

	return false
}

// NewMutex create new redis mutex
func (e *Enity) NewMutex(expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutex(e.conn, expire, checkInterval)
}

// NewMutexByID create new redis mutex with selected id
func (e *Enity) NewMutexByID(lockID string, expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(e.conn, lockID, expire, checkInterval)
}

// GetMetrics return map of the metrics from cache connection
func (e *Enity) GetMetrics(ctx context.Context) (base.MapMetricsOptions, error) {
	e.Metrics.AddNewMetricGauge(
		e.GetFullName(),
		"status",
		utils.JoinStrings(" ", "status link to", utils.RedactedDSN(e.cfg.DSN)),
		func() float64 {
			if e.conn != nil {
				err := e.ping(ctx)
				if err == nil {
					return 1
				}
			}
			return 0
		},
	)

	e.Metrics.AddNewMetricGauge(
		e.GetFullName(),
		"cache size",
		"cache size",
		func() float64 {
			size, _ := e.Size(ctx)
			return float64(size)
		},
	)

	return e.Metrics, nil
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	e.ReadyHandlers[strings.ToUpper(e.GetFullName()+"_notfailed")] = func() (bool, string) {
		if e.conn == nil {
			return false, "not connected"
		}

		if err := e.ping(ctx); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return e.ReadyHandlers, nil
}
