package redis

import (
	"context"
	"strings"
	"time"

	"github.com/KromDaniel/rejonson"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	ctx     context.Context
	log     zerolog.Logger
	name    string
	options *Config

	Conn *rejonson.Client

	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool

	// Metrics
	stats.Service
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, opts interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	conn := &Enity{
		name: name,
		ctx:  logger.Registrate(ctx),
	}

	logger.Get(conn.ctx).GetLogger(cache.ProvidersName, nil).Info().Msgf("initializing enity " + name + "...")
	conn.log = logger.Get(conn.ctx).GetLogger(cache.ProvidersName, nil).With().Str("connection", name).Logger()

	// We should not attempt to establish connection if passed options
	// isn't OUR options.
	var ok bool
	conn.options, ok = opts.(*Config)
	if !ok {
		return nil, cache.ErrInvalidConnectionOptionsPointer(Config{})
	}

	conn.options.Validate()
	conn.connWatcherStopped = true

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (conn *Enity) Shutdown() error {
	conn.log.Info().Msg("shutting down database connection watcher and queue worker")
	conn.weAreShuttingDown = true

	for {
		if conn.connWatcherStopped {
			break
		}

		time.Sleep(time.Millisecond * 500)
	}
	conn.log.Info().Msg("connection shutted down")

	return nil
}

// SetConnPoolLifetime sets connection lifetime.
func (conn *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	conn.options.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if conn.Conn != nil {
		conn.Conn.Options().MaxConnAge = connMaxLifetime
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (conn *Enity) SetConnPoolLimits(minIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	conn.options.MinIdleConnections = minIdleConnections
	conn.options.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if conn.Conn != nil {
		conn.Conn.Options().MinIdleConns = minIdleConnections
		conn.Conn.Options().PoolSize = maxOpenedConnections
	}
}

// SetPoolLimits sets connection pool limits.
func (conn *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	conn.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	conn.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (conn *Enity) Start() error {
	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if conn.connWatcherStopped {
		if conn.options.StartWatcher {
			conn.connWatcherStopped = false
			go conn.startWatcher()
		} else {
			// Manually start the connection once to establish connection
			_ = conn.watcher()
		}
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established.
func (conn *Enity) WaitForEstablishing() {
	for {
		if conn.Conn != nil {
			break
		}

		conn.log.Debug().Msg("connection isn't ready - not yet established")
		time.Sleep(time.Millisecond * 100)
	}
}

// Get item from cache by key.
func (conn *Enity) Get(key string, value interface{}) error {
	cmdString := conn.Conn.Get(conn.options.KeyPrefix + key)
	_, err := cmdString.Result()

	if err != nil {
		return cache.ErrNotFoundInCache
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	conn.log.Debug().Msgf("get value by key %s from cache", key)

	return nil
}

// JSONGet item from cache by key.
func (conn *Enity) JSONGet(key, path string, value interface{}) error {
	cmdString := conn.Conn.JsonGet(conn.options.KeyPrefix+key, path)
	_, err := cmdString.Result()

	if err != nil {
		return cache.ErrNotFoundInCache
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	conn.log.Debug().Msgf("JSONGet value by key %s from cache, path %s", key, path)

	return nil
}

// Set item in cache by key.
func (conn *Enity) Set(key string, value interface{}) error {
	_, err := conn.Conn.Set(conn.options.KeyPrefix+key, value, conn.options.ClearTime).Result()
	if err != nil {
		return err
	}

	conn.log.Debug().Msgf("set key %s in cache, value %+v", key, value)

	return nil
}

// SetNX (Not eXist) item in cache by key.
func (conn *Enity) SetNX(key string, value interface{}) error {
	_, err := conn.Conn.SetNX(conn.options.KeyPrefix+key, value, conn.options.ClearTime).Result()
	if err != nil {
		return err
	}

	conn.log.Debug().Msgf("SetNX key %s in cache, value %+v", key, value)

	return nil
}

// JSONSet item in cache by key.
func (conn *Enity) JSONSet(key, path, json string) error {
	_, err := conn.Conn.JsonSet(conn.options.KeyPrefix+key, path, json).Result()
	if err != nil {
		return err
	}

	if conn.options.ClearTime > 0 {
		_, err = conn.Conn.Expire(conn.options.KeyPrefix+key, conn.options.ClearTime).Result()
		if err != nil {
			return err
		}
	}

	conn.log.Debug().Msgf("JSONSet key %s in cache, path %s, json %s", key, path, json)

	return nil
}

// JSONSetNX item in cache by key.
func (conn *Enity) JSONSetNX(key, path, json string) error {
	_, err := conn.Conn.JsonSet(conn.options.KeyPrefix+key, path, json, "NX").Result()
	if err != nil {
		return err
	}

	if conn.options.ClearTime > 0 {
		_, err = conn.Conn.Expire(conn.options.KeyPrefix+key, conn.options.ClearTime).Result()
		if err != nil {
			return err
		}
	}

	conn.log.Debug().Msgf("JSONSetNX key %s in cache, path %s, json %s", key, path, json)

	return nil
}

// Delete item from cache by key.
func (conn *Enity) Delete(key string) error {
	_, err := conn.Conn.Del(conn.options.KeyPrefix + key).Result()
	if err != nil {
		return err
	}

	conn.log.Debug().Msgf("delete key %s from cache", key)

	return nil
}

// Size return count of item in cache
func (conn *Enity) Size() int {
	length := 0
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = conn.Conn.Scan(cursor, conn.options.KeyPrefix+"*", 10).Result()
		if err != nil {
			return -1
		}

		length += len(keys)

		if cursor == 0 {
			break
		}
	}

	conn.log.Debug().Msgf("cache size is %d", length)

	return length
}

// Clear clear all items from selected connection.
func (conn *Enity) Clear() error {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = conn.Conn.Scan(cursor, conn.options.KeyPrefix+"*", 10).Result()
		if err != nil {
			return err
		}

		pipe := conn.Conn.Pipeline()
		err = pipe.Del(keys...).Err()
		if err != nil {
			return err
		}

		_, err = pipe.Exec()
		if err != nil {
			return err
		}
		if cursor == 0 {
			break
		}
	}

	conn.log.Debug().Msgf("delete all keys from cache")

	return nil
}

// NewMutex create new redis mutex
func (conn *Enity) NewMutex(expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutex(&conn.Conn, expire, checkInterval)
}

// NewMutexByID create new redis mutex with selected id
func (conn *Enity) NewMutexByID(lockID interface{}, expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(&conn.Conn, lockID, expire, checkInterval)
}

// GetMetrics return map of the metrics from cache connection
func (conn *Enity) GetMetrics(prefix string) stats.MapMetricsOptions {
	_ = conn.Service.GetMetrics(prefix)
	conn.Metrics[prefix+"_"+conn.name+"_status"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + conn.name + "_status",
				Help: prefix + " " + conn.name + " status link to " + utils.RedactedDSN(conn.options.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			if conn.Conn != nil {
				_, err := conn.Conn.Ping().Result()
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}

	conn.Metrics[prefix+"_"+conn.name+"_cache_size"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + conn.name + "_cache_size",
				Help: prefix + " " + conn.name + " cache size",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(conn.Size()))
		},
	}

	return conn.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (conn *Enity) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	_ = conn.Service.GetReadyHandlers(prefix)
	conn.ReadyHandlers[strings.ToUpper(prefix+"_"+conn.name+"_notfailed")] = func() (bool, string) {
		if conn.Conn == nil {
			return false, "not connected"
		}

		if _, err := conn.Conn.Ping().Result(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return conn.ReadyHandlers
}
